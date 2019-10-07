# AST log

Just like `git log -p` but for an ast node (function, method, class, ...).

The difference from `git log -L start,end:file` it traces changes based on tree diff instead of text diff.

### Usage

```
ast-log -r <repo-path> -f <file-path>
# find node id in the output

ast-log -r <repo-path> -f <file-path> --id <node-id>
```

### Dependencies

Currently the command depends on [bblfshd](https://github.com/bblfsh/bblfshd/) and expects the server running on `0.0.0.0:9432`.

### Example output

```diff
$ go run main.go -r /Users/smacker/go/src/gopkg.in/src-d/go-git.v4 -f repository.go --id 2611
commit 02335b10dee417d0338bf6ea070feeead18e636b
Author: Jeremy Chambers <jeremy@thehipbot.com>
Date:   Sat Apr 07 14:34:39 2018 -0500

    config: adds branches to config for tracking branches against remotes, updates clone to track when cloning a branch. Fixes #313

    Signed-off-by: Jeremy Chambers <jeremy@thehipbot.com>


--- Original
+++ Current
@@ -51,5 +51,27 @@
 		}
 	}

-	return r.updateRemoteConfigIfNeeded(o, c, ref)
+	if err := r.updateRemoteConfigIfNeeded(o, c, ref); err != nil {
+		return err
+	}
+
+	if ref.Name().IsBranch() {
+		branchRef := ref.Name()
+		branchName := strings.Split(string(branchRef), "refs/heads/")[1]
+
+		b := &config.Branch{
+			Name:  branchName,
+			Merge: branchRef,
+		}
+		if o.RemoteName == "" {
+			b.Remote = "origin"
+		} else {
+			b.Remote = o.RemoteName
+		}
+		if err := r.CreateBranch(b); err != nil {
+			return err
+		}
+	}
+
+	return nil
 }

commit 591aed138177b27b08a90c90e6e074a6cf2dbd00
Author: Michael Rykov <mrykov@gmail.com>
Date:   Mon Jan 15 12:32:53 2018 -0800

    Support for clone without checkout

--- Original
+++ Current
@@ -23,7 +23,7 @@
 		return err
 	}

-	if r.wt != nil {
+	if r.wt != nil && !o.NoCheckout {
 		w, err := r.Worktree()
 		if err != nil {
 			return err

commit caa4dc4729e50e3a10ccd438d02d69fa20f9b766
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Thu Nov 23 07:19:39 2017 +0100

    update to go-billy.v4 and go-git-fixtures.v3

    Signed-off-by: Máximo Cuadros <mcuadros@gmail.com>


--- Original
+++ Current
@@ -12,11 +12,12 @@
 		return err
 	}

-	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
+	ref, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
 		RefSpecs: r.cloneRefSpec(o, c),
 		Depth:    o.Depth,
 		Auth:     o.Auth,
 		Progress: o.Progress,
+		Tags:     o.Tags,
 	}, o.ReferenceName)
 	if err != nil {
 		return err
@@ -28,7 +29,15 @@
 			return err
 		}

-		if err := w.Reset(&ResetOptions{Commit: head.Hash()}); err != nil {
+		head, err := r.Head()
+		if err != nil {
+			return err
+		}
+
+		if err := w.Reset(&ResetOptions{
+			Mode:   MergeReset,
+			Commit: head.Hash(),
+		}); err != nil {
 			return err
 		}

@@ -42,5 +51,5 @@
 		}
 	}

-	return r.updateRemoteConfigIfNeeded(o, c, head)
+	return r.updateRemoteConfigIfNeeded(o, c, ref)
 }

commit e09fa242c1f97547527fa0cb9f6288f9ae17479e
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Mon Nov 20 01:55:48 2017 +0100

    plumbing: object, commit.Parent() method

    Signed-off-by: Máximo Cuadros <mcuadros@gmail.com>


--- Original
+++ Current
@@ -12,12 +12,11 @@
 		return err
 	}

-	ref, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
+	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
 		RefSpecs: r.cloneRefSpec(o, c),
 		Depth:    o.Depth,
 		Auth:     o.Auth,
 		Progress: o.Progress,
-		Tags:     o.Tags,
 	}, o.ReferenceName)
 	if err != nil {
 		return err
@@ -29,15 +28,7 @@
 			return err
 		}

-		head, err := r.Head()
-		if err != nil {
-			return err
-		}
-
-		if err := w.Reset(&ResetOptions{
-			Mode:   MergeReset,
-			Commit: head.Hash(),
-		}); err != nil {
+		if err := w.Reset(&ResetOptions{Commit: head.Hash()}); err != nil {
 			return err
 		}

@@ -51,5 +42,5 @@
 		}
 	}

-	return r.updateRemoteConfigIfNeeded(o, c, ref)
+	return r.updateRemoteConfigIfNeeded(o, c, head)
 }

commit 7cdc44306dd1b3bba4a219bf3c40c5097a505a8e
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Mon Sep 04 13:54:02 2017 +0200

    Repository.Clone added Tags option, and set by default AllTags as git does


--- Original
+++ Current
@@ -17,6 +17,7 @@
 		Depth:    o.Depth,
 		Auth:     o.Auth,
 		Progress: o.Progress,
+		Tags:     o.Tags,
 	}, o.ReferenceName)
 	if err != nil {
 		return err

commit f1e58e0d30095cf768ff04d379b5e4145a874be8
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Fri Sep 01 17:26:52 2017 +0200

    Worktree.Reset ignore untracked files on Merge mode


--- Original
+++ Current
@@ -12,7 +12,7 @@
 		return err
 	}

-	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
+	ref, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
 		RefSpecs: r.cloneRefSpec(o, c),
 		Depth:    o.Depth,
 		Auth:     o.Auth,
@@ -28,7 +28,15 @@
 			return err
 		}

-		if err := w.Reset(&ResetOptions{Commit: head.Hash()}); err != nil {
+		head, err := r.Head()
+		if err != nil {
+			return err
+		}
+
+		if err := w.Reset(&ResetOptions{
+			Mode:   MergeReset,
+			Commit: head.Hash(),
+		}); err != nil {
 			return err
 		}

@@ -42,5 +50,5 @@
 		}
 	}

-	return r.updateRemoteConfigIfNeeded(o, c, head)
+	return r.updateRemoteConfigIfNeeded(o, c, ref)
 }

commit 5a7b7af0f793b1c25e9543e8511b767f3b739d67
Author: Miguel Molina <miguel@erizocosmi.co>
Date:   Thu Aug 24 17:35:38 2017 +0200

    dotgit: rewrite the way references are looked up
    Now there's only two ways of getting a reference, by checking under refs/ directory or in packed-refs. refs/ directory is checked using a direct read by reference name and packed refs are cached until they have been changed.

    Signed-off-by: Miguel Molina <miguel@erizocosmi.co>


--- Original
+++ Current
@@ -12,7 +12,7 @@
 		return err
 	}

-	ref, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
+	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
 		RefSpecs: r.cloneRefSpec(o, c),
 		Depth:    o.Depth,
 		Auth:     o.Auth,
@@ -24,11 +24,6 @@

 	if r.wt != nil {
 		w, err := r.Worktree()
-		if err != nil {
-			return err
-		}
-
-		head, err := r.Head()
 		if err != nil {
 			return err
 		}
@@ -47,5 +42,5 @@
 		}
 	}

-	return r.updateRemoteConfigIfNeeded(o, c, ref)
+	return r.updateRemoteConfigIfNeeded(o, c, head)
 }

commit 17cde59e5ced61adece4741b3a4da947f08fd9dc
Author: Ori Rawlings <orirawlings@gmail.com>
Date:   Wed Aug 23 22:39:25 2017 -0500

    repository: Resolve commit when cloning annotated tag, fixes #557


--- Original
+++ Current
@@ -12,7 +12,7 @@
 		return err
 	}

-	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
+	ref, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
 		RefSpecs: r.cloneRefSpec(o, c),
 		Depth:    o.Depth,
 		Auth:     o.Auth,
@@ -24,6 +24,11 @@

 	if r.wt != nil {
 		w, err := r.Worktree()
+		if err != nil {
+			return err
+		}
+
+		head, err := r.Head()
 		if err != nil {
 			return err
 		}
@@ -42,5 +47,5 @@
 		}
 	}

-	return r.updateRemoteConfigIfNeeded(o, c, head)
+	return r.updateRemoteConfigIfNeeded(o, c, ref)
 }

commit a0b45cc5508ae48b01799ca800e464888ed598be
Author: Josh Bleecher Snyder <josharian@gmail.com>
Date:   Thu Aug 03 16:46:57 2017 -0700

    plumbing/object: add Commit.FirstParent

    First parents are somewhat special in git.
    There's even a --first-parent flag to 'git log'.

    Add a helper method to look them up.
    This avoids boilerplate and spares the client from
    having to arrange for a handle to the Storer,
    which is stored in the unexported field Commit.s.




--- Original
+++ Current
@@ -5,7 +5,7 @@

 	c := &config.RemoteConfig{
 		Name: o.RemoteName,
-		URL:  o.URL,
+		URLs: []string{o.URL},
 	}

 	if _, err := r.CreateRemote(c); err != nil {

commit c128f5d680f59fd125cafd90f10e39eae5f3a135
Author: Jeremy Stribling <strib@keyba.se>
Date:   Mon Jul 31 15:34:45 2017 -0700

    plumbing: fix pack commands for the file client on Windows

    The default git install on Windows doesn't come with commands for
    receive-pack and upload-pack in the default $PATH.  Instead, use
    --exec-path to find pack executables in that case.


--- Original
+++ Current
@@ -5,7 +5,7 @@

 	c := &config.RemoteConfig{
 		Name: o.RemoteName,
-		URLs: []string{o.URL},
+		URL:  o.URL,
 	}

 	if _, err := r.CreateRemote(c); err != nil {

commit b29ccd9cf64cb3c6d7b3fdc6649d97416f3be734
Author: Manuel Carmona <manu.carmona90@gmail.com>
Date:   Thu Aug 03 09:41:22 2017 +0200

    *: windows support, some more fixes (#533)

    * fixed windows failed test: "134 FAIL: repository_test.go:340: RepositorySuite.TestPlainOpenBareRelativeGitDirFileTrailingGarbage"

    * fixed windows failed test: "143 FAIL: worktree_test.go:367: WorktreeSuite.TestCheckoutIndexOS"

    * fixed windows failed test: "296 FAIL: receive_pack_test.go:36: ReceivePackSuite.TearDownTest"

    * fixed windows failed test: "152 FAIL: worktree_test.go:278: WorktreeSuite.TestCheckoutSymlink"


--- Original
+++ Current
@@ -5,7 +5,7 @@

 	c := &config.RemoteConfig{
 		Name: o.RemoteName,
-		URL:  o.URL,
+		URLs: []string{o.URL},
 	}

 	if _, err := r.CreateRemote(c); err != nil {

commit 5c1a2ec798eb9b78d66b16fbbcbdc3b928d8b496
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Wed Aug 02 17:28:02 2017 +0200

    worktree: normalized string comparison tests


--- Original
+++ Current
@@ -5,7 +5,7 @@

 	c := &config.RemoteConfig{
 		Name: o.RemoteName,
-		URLs: []string{o.URL},
+		URL:  o.URL,
 	}

 	if _, err := r.CreateRemote(c); err != nil {

commit 595de2b38d0cee2e0bc92e1a0559f16ccca851dc
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Wed Aug 02 14:26:58 2017 +0200

    Remote.Clone fix clone of tags in shallow mode


--- Original
+++ Current
@@ -5,7 +5,7 @@

 	c := &config.RemoteConfig{
 		Name: o.RemoteName,
-		URL:  o.URL,
+		URLs: []string{o.URL},
 	}

 	if _, err := r.CreateRemote(c); err != nil {
@@ -33,7 +33,10 @@
 		}

 		if o.RecurseSubmodules != NoRecurseSubmodules {
-			if err := w.updateSubmodules(o.RecurseSubmodules); err != nil {
+			if err := w.updateSubmodules(&SubmoduleUpdateOptions{
+				RecurseSubmodules: o.RecurseSubmodules,
+				Auth:              o.Auth,
+			}); err != nil {
 				return err
 			}
 		}

commit 171b3a73e7ab7015f9eb8e98965e36dfb8ea9599
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Wed Aug 02 13:00:31 2017 +0200

    plumbing: moved `Reference.Is*` methods to `ReferenceName.Is*`


--- Original
+++ Current
@@ -5,7 +5,7 @@

 	c := &config.RemoteConfig{
 		Name: o.RemoteName,
-		URLs: []string{o.URL},
+		URL:  o.URL,
 	}

 	if _, err := r.CreateRemote(c); err != nil {

commit 9488c59834f6a2591910b7b360721cec2c16c548
Author: Santiago M. Mola <santi@mola.io>
Date:   Mon Jul 24 10:51:01 2017 +0200

    config: multiple values in RemoteConfig (URLs and Fetch)

    * Change `URL string` to `URL []string` in `RemoteConfig`, since
      git allows multiple URLs per remote. See:
      http://marc.info/?l=git&m=116231242118202&w=2

    * Fix marshalling of multiple fetch refspecs.


--- Original
+++ Current
@@ -5,7 +5,7 @@

 	c := &config.RemoteConfig{
 		Name: o.RemoteName,
-		URL:  o.URL,
+		URLs: []string{o.URL},
 	}

 	if _, err := r.CreateRemote(c); err != nil {
@@ -33,10 +33,7 @@
 		}

 		if o.RecurseSubmodules != NoRecurseSubmodules {
-			if err := w.updateSubmodules(&SubmoduleUpdateOptions{
-				RecurseSubmodules: o.RecurseSubmodules,
-				Auth:              o.Auth,
-			}); err != nil {
+			if err := w.updateSubmodules(o.RecurseSubmodules); err != nil {
 				return err
 			}
 		}

commit 63b30fba572b7e70833fae4785c6d22f167c6641
Author: Devon Barrett <devon@devonbarrett.net>
Date:   Sat Jul 29 14:12:57 2017 +0200

    reuse Auth method when recursing submodules, fixes #521


--- Original
+++ Current
@@ -33,7 +33,10 @@
 		}

 		if o.RecurseSubmodules != NoRecurseSubmodules {
-			if err := w.updateSubmodules(o.RecurseSubmodules); err != nil {
+			if err := w.updateSubmodules(&SubmoduleUpdateOptions{
+				RecurseSubmodules: o.RecurseSubmodules,
+				Auth:              o.Auth,
+			}); err != nil {
 				return err
 			}
 		}

commit ab590eb89849a0319b8c5a4d7fd980137da7180d
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Wed Jul 26 21:46:49 2017 +0200

    worktree: expose underlying filesystem


--- Original
+++ Current
@@ -1,4 +1,4 @@
-func (r *Repository) clone(o *CloneOptions) error {
+func (r *Repository) clone(ctx context.Context, o *CloneOptions) error {
 	if err := o.Validate(); err != nil {
 		return err
 	}
@@ -12,7 +12,7 @@
 		return err
 	}

-	head, err := r.fetchAndUpdateReferences(&FetchOptions{
+	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
 		RefSpecs: r.cloneRefSpec(o, c),
 		Depth:    o.Depth,
 		Auth:     o.Auth,

commit c64eb817d5e5cbaec10dea1342e1ec95721e886b
Author: Santiago M. Mola <santi@mola.io>
Date:   Tue Jul 25 10:08:36 2017 +0200

    packfile: create packfile.Index and reuse it

    There was an internal type (i.e. storage/filesystem.idx) to
    use as in-memory index for packfiles. This was not convenient
    to reuse in the packfile.

    This commit creates a new representation (format/packfile.Index)
    that can be converted to and from idxfile.Idxfile.

    A packfile.Index now contains the functionality that was scattered
    on storage/filesystem.idx and packfile.Decoder's internals.

    storage/filesystem now reuses packfile.Index instances and this
    also results in higher cache hit ratios when resolving deltas.


--- Original
+++ Current
@@ -1,4 +1,4 @@
-func (r *Repository) clone(ctx context.Context, o *CloneOptions) error {
+func (r *Repository) clone(o *CloneOptions) error {
 	if err := o.Validate(); err != nil {
 		return err
 	}
@@ -12,7 +12,7 @@
 		return err
 	}

-	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
+	head, err := r.fetchAndUpdateReferences(&FetchOptions{
 		RefSpecs: r.cloneRefSpec(o, c),
 		Depth:    o.Depth,
 		Auth:     o.Auth,

commit 064051f972e90dd55e6c941f04b58b4a36dfedf1
Author: Máximo Cuadros <mcuadros@gmail.com>
Date:   Wed Jul 26 06:24:47 2017 +0200

    *: package context support in Repository, Remote and Submodule


--- Original
+++ Current
@@ -1 +1,43 @@
+func (r *Repository) clone(ctx context.Context, o *CloneOptions) error {
+	if err := o.Validate(); err != nil {
+		return err
+	}

+	c := &config.RemoteConfig{
+		Name: o.RemoteName,
+		URL:  o.URL,
+	}
+
+	if _, err := r.CreateRemote(c); err != nil {
+		return err
+	}
+
+	head, err := r.fetchAndUpdateReferences(ctx, &FetchOptions{
+		RefSpecs: r.cloneRefSpec(o, c),
+		Depth:    o.Depth,
+		Auth:     o.Auth,
+		Progress: o.Progress,
+	}, o.ReferenceName)
+	if err != nil {
+		return err
+	}
+
+	if r.wt != nil {
+		w, err := r.Worktree()
+		if err != nil {
+			return err
+		}
+
+		if err := w.Reset(&ResetOptions{Commit: head.Hash()}); err != nil {
+			return err
+		}
+
+		if o.RecurseSubmodules != NoRecurseSubmodules {
+			if err := w.updateSubmodules(o.RecurseSubmodules); err != nil {
+				return err
+			}
+		}
+	}
+
+	return r.updateRemoteConfigIfNeeded(o, c, head)
+}
```

### Known issues & TODO

- Performance

    It works slow as hell. Timing for the log above:

    ```
    Total time	43.952412822s	100%
    Go-git operations time	11.821716895s	26%
    Bblfsh parsing time	28.499216239s	64%
    Gum matching time	3.566183643s	8%
    ```

    Possible solution could be to switch to tree-sitter which is _much_ faster than bblfsh and the tool can take advantage of incremental parsing. Then take a look why go-git operations are so slow and optimize them.

- Lack of human interface for choosing target node

    Possible solution could be to accept line:char and suggest to choose a node on at this position.

- Incorrect matching 

    In some cases the node can be match "incorrectly" (according to human expectations).

    Possible solution could be to tune ast diff algorithm for this particular case.

### Thanks

- Jean-Rémy Falleri for [GumTreeDiff/gumtree](https://github.com/GumTreeDiff/gumtree) and the paper [Fine-grained and Accurate Source Code Differencing](https://hal.archives-ouvertes.fr/hal-01054552/document).
- Patrick Mézard for [go-difflib](https://github.com/pmezard/go-difflib) library.
- Source{d} for [bblfsh](https://github.com/bblfsh/bblfshd) and [go-git](https://github.com/src-d/go-git).
