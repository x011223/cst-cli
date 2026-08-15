package git

import "strings"

// TreeNode is a node in a repository's changed-file tree.
type TreeNode struct {
	Name     string
	Status   string // raw git status, empty for directories
	IsFile   bool
	Children []*TreeNode
}

// SortedChildren returns children ordered directories-first, then files,
// alphabetical within each group.
func (n *TreeNode) SortedChildren() []*TreeNode {
	kids := append([]*TreeNode(nil), n.Children...)
	for i := 1; i < len(kids); i++ {
		for j := i; j > 0 && less(kids[j], kids[j-1]); j-- {
			kids[j], kids[j-1] = kids[j-1], kids[j]
		}
	}
	return kids
}

func less(a, b *TreeNode) bool {
	if a.IsFile != b.IsFile {
		return !a.IsFile // directories come first
	}
	return a.Name < b.Name
}

// buildTree turns the repository changes into a nested tree keyed by path parts.
func buildTree(changes []Change) *TreeNode {
	root := &TreeNode{Children: []*TreeNode{}}
	for _, c := range changes {
		parts := strings.Split(c.Path, "/")
		cur := root
		for i, p := range parts {
			var child *TreeNode
			for _, ch := range cur.Children {
				if ch.Name == p {
					child = ch
					break
				}
			}
			if child == nil {
				child = &TreeNode{Name: p, Children: []*TreeNode{}}
				cur.Children = append(cur.Children, child)
			}
			if i == len(parts)-1 {
				child.IsFile = true
				child.Status = c.Status
			}
			cur = child
		}
	}
	return root
}

// Tree returns the repository changes as a nested tree.
func (r RepoStatus) Tree() *TreeNode { return buildTree(r.Changes) }
