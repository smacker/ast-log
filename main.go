package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/sirupsen/logrus"
	"github.com/smacker/gum"
	"github.com/smacker/gum/uast"
	bblfsh "gopkg.in/bblfsh/client-go.v3"
	bblfshUAST "gopkg.in/bblfsh/sdk.v2/uast"
	"gopkg.in/bblfsh/sdk.v2/uast/nodes"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type options struct {
	Repo   string `short:"r" long:"repository-path" description:"Path to git repository" default:"."`
	File   string `short:"f" long:"file-path" description:"Path to the file to diff" required:"true"`
	NodeID int    `long:"id" description:"Node id number"`
	Debug  bool   `long:"debug" description:"Enable debug logging"`
	Timing bool   `long:"timing" description:"Print timing"`
}

var opts options
var parser = flags.NewParser(&opts, flags.Default)
var logger = logrus.New()
var sts = newStats()

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			parser.WriteHelp(os.Stderr)
			os.Exit(1)
		}
	}

	if opts.Debug {
		logger.SetLevel(logrus.DebugLevel)
	}

	if opts.NodeID == 0 {
		fmt.Println("Choose node id")
		if err := printFile(opts.Repo, opts.File); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	if err := run(opts.Repo, opts.File, opts.NodeID); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if opts.Timing {
		sts.Print()
	}
}

func printFile(repoPath, filePath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	head, err := repo.Head()
	if err != nil {
		return err
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return err
	}

	content, err := getContent(commit, filePath)
	if err != nil {
		return err
	}

	t, err := parseFile(filePath, content)
	if err != nil {
		return err
	}

	printTree(t, 0)
	return nil
}

func printTree(t *gum.Tree, tab int) {
	fmt.Println(strings.Repeat("-", tab), t, t.GetID())
	for _, c := range t.Children {
		printTree(c, tab+1)
	}
}

func run(repoPath, path string, nodeID int) error {
	defer sts.running()()

	commits, err := commitsWithDiff(repoPath, path, nodeID)
	if err != nil {
		return err
	}

	for _, commit := range commits {
		fmt.Println(commit.String())

		var srcNodeContent string
		if commit.srcNode != nil {
			srcNodeContent = getNodeContent(commit.srcNode, commit.srcContent)
		}
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(srcNodeContent),
			B:        difflib.SplitLines(getNodeContent(commit.dstNode, commit.dstContent)),
			FromFile: "Original",
			ToFile:   "Current",
			Context:  3,
		}
		text, _ := difflib.GetUnifiedDiffString(diff)
		fmt.Println(text)
	}

	return nil
}

func findCommits(repoPath, filePath string) ([]*object.Commit, error) {
	defer sts.gitting()()

	var results []*object.Commit

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	commits, err := repo.Log(&git.LogOptions{
		From:     head.Hash(),
		Order:    git.LogOrderCommitterTime,
		FileName: &filePath,
	})
	if err != nil {
		return nil, err
	}

	for {
		c, err := commits.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// skip merge commits
		if len(c.ParentHashes) > 1 {
			continue
		}

		results = append(results, c)
	}

	return results, nil
}

func getContent(c *object.Commit, filename string) ([]byte, error) {
	defer sts.gitting()()

	f, err := c.File(filename)
	if err != nil {
		return nil, err
	}

	r, err := f.Reader()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(r)
}

func parseFile(path string, content []byte) (*gum.Tree, error) {
	defer sts.parsing()()

	client, err := bblfsh.NewClient("0.0.0.0:9432")
	if err != nil {
		return nil, fmt.Errorf("can't connect to bblfsh: %s", err)
	}
	res, _, err := client.
		NewParseRequest().
		Mode(bblfsh.Annotated).
		Filename(path).
		Content(string(content)).
		UAST()
	if err != nil {
		return nil, fmt.Errorf("can't parse the file %s: %s", path, err)
	}
	return uast.ToTree(res), nil
}

func nodeOffset(t *gum.Tree) (int, int) {
	n := t.Meta.(nodes.Node)
	pos := bblfshUAST.PositionsOf(n)
	start := int(pos["start"].Offset)
	end := int(pos["end"].Offset)

	return start, end
}

type diff struct {
	*object.Commit

	srcContent []byte
	dstContent []byte

	srcTree  *gum.Tree
	dstTree  *gum.Tree
	mappings []gum.Mapping

	srcNode *gum.Tree
	dstNode *gum.Tree
}

func commitsWithDiff(repoPath, path string, nodeID int) ([]*diff, error) {
	var result []*diff

	commits, err := findCommits(repoPath, path)
	if err != nil {
		return nil, err
	}

	var lastCommit *object.Commit
	var lastContent []byte
	var dstTree *gum.Tree
	var dstNode *gum.Tree
	for _, commit := range commits {
		log := logger.WithField("commit", commit.Hash.String()).WithField("title", strings.TrimSpace(commit.Message))

		content, err := getContent(commit, path)
		if err != nil {
			return nil, err
		}
		srcTree, err := parseFile(path, content)
		if err != nil {
			return nil, err
		}
		//log.Debug("file parsed")

		if dstTree == nil {
			for _, t := range gum.PostOrder(srcTree) {
				if t.GetID() != nodeID {
					continue
				}

				dstNode = t
				break
			}

			if dstNode == nil {
				return nil, fmt.Errorf("node with id %d not found", nodeID)
			}
			log.Debug("target node is set")
			log.Debugf("Node content:\n%s", getNodeContent(dstNode, content))

			dstTree = srcTree
			lastContent = content
			lastCommit = commit
			continue
		}

		endMatching := sts.matching()
		var srcNode *gum.Tree
		mappings := gum.Match(srcTree, dstTree)
		for _, m := range mappings {
			if m[1] != dstNode {
				continue
			}
			srcNode = m[0]
		}
		endMatching()

		if srcNode != nil && dstNode.IsIsomorphicTo(srcNode) {
			log.Debug("found isomorphic node, skipping commit")

			lastCommit = commit
			continue
		}
		if srcNode != nil {
			log.Debug("found matched node")
			log.Debugf("Node content:\n%s", getNodeContent(srcNode, content))
		}

		result = append(result, &diff{
			Commit: lastCommit,

			srcContent: content[:],
			dstContent: lastContent[:],

			srcTree:  srcTree,
			dstTree:  dstTree,
			mappings: mappings,

			srcNode: srcNode,
			dstNode: dstNode,
		})

		dstTree = srcTree
		dstNode = srcNode
		lastContent = content
		lastCommit = commit

		if srcNode == nil {
			log.Debug("can't find node in commit")
			return result, nil
		}
	}

	result = append(result, &diff{
		Commit: lastCommit,

		srcContent: nil,
		dstContent: lastContent[:],

		srcTree:  nil,
		dstTree:  dstTree,
		mappings: nil,

		srcNode: nil,
		dstNode: dstNode,
	})

	return result, nil
}

func getNodeContent(n *gum.Tree, content []byte) string {
	start, end := nodeOffset(n)
	return string(content[start:end])
}

//

type stats struct {
	total         time.Duration
	goGit         time.Duration
	bblfshParsing time.Duration
	gumMatching   time.Duration
}

func newStats() *stats {
	return &stats{}
}

func (s *stats) running() func() {
	start := time.Now()
	return func() {
		s.total += time.Now().Sub(start)
	}
}

func (s *stats) gitting() func() {
	start := time.Now()
	return func() {
		s.goGit += time.Now().Sub(start)
	}
}

func (s *stats) parsing() func() {
	start := time.Now()
	return func() {
		s.bblfshParsing += time.Now().Sub(start)
	}
}

func (s *stats) matching() func() {
	start := time.Now()
	return func() {
		s.gumMatching += time.Now().Sub(start)
	}
}

func (s *stats) Print() {
	fmt.Printf("Total time\t%s\t%d%%\n", s.total, 100)
	fmt.Printf("Go-git operations time\t%s\t%d%%\n", s.goGit, int(s.goGit*100/s.total))
	fmt.Printf("Bblfsh parsing time\t%s\t%d%%\n", s.bblfshParsing, int(s.bblfshParsing*100/s.total))
	fmt.Printf("Gum matching time\t%s\t%d%%\n", s.gumMatching, int(s.gumMatching*100/s.total))
}
