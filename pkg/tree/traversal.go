package tree

import (
	"fmt"
	"sort"
	"strings"
)

// Flatten returns all nodes in the tree in pre-order traversal (depth-first).
func Flatten(nodes []*Node) []*Node {
	var result []*Node
	for _, n := range nodes {
		result = append(result, n)
		result = append(result, Flatten(n.Children)...)
	}
	return result
}

// LeafNodes returns only the leaf nodes from the tree.
func LeafNodes(nodes []*Node) []*Node {
	var result []*Node
	for _, n := range nodes {
		if n.IsLeaf() {
			result = append(result, n)
		} else {
			result = append(result, LeafNodes(n.Children)...)
		}
	}
	return result
}

// FindByID searches the tree for a node with the given ID.
func FindByID(nodes []*Node, id string) *Node {
	for _, n := range nodes {
		if n.NodeID == id {
			return n
		}
		if found := FindByID(n.Children, id); found != nil {
			return found
		}
	}
	return nil
}

// FindByTitle searches the tree for a node with the given title (case-insensitive).
func FindByTitle(nodes []*Node, title string) *Node {
	lower := strings.ToLower(title)
	for _, n := range nodes {
		if strings.ToLower(n.Title) == lower {
			return n
		}
		if found := FindByTitle(n.Children, title); found != nil {
			return found
		}
	}
	return nil
}

// AssignNodeIDs assigns zero-padded sequential IDs to all nodes in pre-order.
func AssignNodeIDs(nodes []*Node) {
	all := Flatten(nodes)
	width := 4
	for i, n := range all {
		n.NodeID = fmt.Sprintf("%0*d", width, i)
	}
}

// StripText removes text from all nodes (useful for structure-only views).
func StripText(nodes []*Node) []*Node {
	result := make([]*Node, len(nodes))
	for i, n := range nodes {
		c := *n
		c.Text = ""
		if len(n.Children) > 0 {
			c.Children = StripText(n.Children)
		}
		result[i] = &c
	}
	return result
}

// StripSummary removes summaries from all nodes.
func StripSummary(nodes []*Node) []*Node {
	result := make([]*Node, len(nodes))
	for i, n := range nodes {
		c := *n
		c.Summary = ""
		c.PrefixSummary = ""
		if len(n.Children) > 0 {
			c.Children = StripSummary(n.Children)
		}
		result[i] = &c
	}
	return result
}

// Depth returns the maximum depth of the tree (root nodes are depth 1).
func Depth(nodes []*Node) int {
	maxD := 0
	for _, n := range nodes {
		d := 1
		if childDepth := Depth(n.Children); childDepth > 0 {
			d += childDepth
		}
		if d > maxD {
			maxD = d
		}
	}
	return maxD
}

// ListToTree converts a flat list of TOCEntry items into a nested tree of Nodes
// using structure codes (e.g., "1", "1.1", "1.2", "2").
func ListToTree(entries []TOCEntry) []*Node {
	if len(entries) == 0 {
		return nil
	}

	// Sort by structure code
	sorted := make([]TOCEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return compareStructureCodes(sorted[i].Structure, sorted[j].Structure) < 0
	})

	var roots []*Node
	// Stack holds (node, level) pairs
	type stackItem struct {
		node  *Node
		level int
	}
	var stack []stackItem

	for i, entry := range sorted {
		level := structureLevel(entry.Structure)
		node := &Node{
			Title:      entry.Title,
			StartIndex: entry.PhysicalIndex,
		}

		// Calculate end index from next entry at same or higher level
		node.EndIndex = calcEndIndex(sorted, i)

		// Pop stack until we find a parent at a lower level
		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, node)
		}

		stack = append(stack, stackItem{node: node, level: level})
	}

	return roots
}

// structureLevel returns the depth level of a structure code.
// "1" -> 1, "1.2" -> 2, "1.2.3" -> 3
func structureLevel(code string) int {
	if code == "" {
		return 0
	}
	return strings.Count(code, ".") + 1
}

// compareStructureCodes compares two structure codes for sorting.
func compareStructureCodes(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int
		if i < len(partsA) {
			fmt.Sscanf(partsA[i], "%d", &numA)
		}
		if i < len(partsB) {
			fmt.Sscanf(partsB[i], "%d", &numB)
		}
		if numA != numB {
			if numA < numB {
				return -1
			}
			return 1
		}
	}

	if len(partsA) < len(partsB) {
		return -1
	}
	if len(partsA) > len(partsB) {
		return 1
	}
	return 0
}

// calcEndIndex calculates the end page index for a TOC entry.
// It looks at the next sibling or parent's next sibling to determine where this node ends.
// The result is guaranteed to be >= the entry's own PhysicalIndex (or 0 if unknown).
func calcEndIndex(entries []TOCEntry, idx int) int {
	currentLevel := structureLevel(entries[idx].Structure)

	for i := idx + 1; i < len(entries); i++ {
		nextLevel := structureLevel(entries[i].Structure)
		if nextLevel <= currentLevel {
			// Next entry at same or higher level = our end
			if entries[i].PhysicalIndex > 0 {
				end := entries[i].PhysicalIndex - 1
				// Ensure end is not below our own start (handles non-monotonic indices)
				if entries[idx].PhysicalIndex > 0 && end < entries[idx].PhysicalIndex {
					end = entries[idx].PhysicalIndex
				}
				return end
			}
		}
	}

	// Last entry or no following sibling — will need to be set to doc end later
	return 0
}

// SetEndIndices fills in missing end indices. Leaf nodes that span to the next
// sibling get their end index set. The last node gets maxPage as end.
// Also enforces EndIndex >= StartIndex for all nodes.
func SetEndIndices(nodes []*Node, maxPage int) {
	flat := Flatten(nodes)
	for i, n := range flat {
		if n.EndIndex == 0 {
			if i+1 < len(flat) && flat[i+1].StartIndex > 0 {
				n.EndIndex = flat[i+1].StartIndex - 1
				if n.EndIndex < n.StartIndex {
					n.EndIndex = n.StartIndex
				}
			} else {
				n.EndIndex = maxPage
			}
		}
		// Enforce EndIndex >= StartIndex even for pre-set values
		// (handles non-monotonic physical indices from LLM mapping)
		if n.StartIndex > 0 && n.EndIndex > 0 && n.EndIndex < n.StartIndex {
			n.EndIndex = n.StartIndex
		}
	}
	// Propagate end indices up: parent end = max of children ends
	propagateEndIndices(nodes)
}

func propagateEndIndices(nodes []*Node) {
	for _, n := range nodes {
		if len(n.Children) > 0 {
			propagateEndIndices(n.Children)
			maxEnd := n.EndIndex
			for _, ch := range n.Children {
				if ch.EndIndex > maxEnd {
					maxEnd = ch.EndIndex
				}
			}
			n.EndIndex = maxEnd
		}
	}
}

// PrintTree returns a text representation of the tree for debugging.
func PrintTree(nodes []*Node, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("%s- %s", prefix, n.Title))
		if n.NodeID != "" {
			sb.WriteString(fmt.Sprintf(" [%s]", n.NodeID))
		}
		if n.StartIndex > 0 {
			sb.WriteString(fmt.Sprintf(" (pp. %d-%d)", n.StartIndex, n.EndIndex))
		}
		sb.WriteString("\n")
		if len(n.Children) > 0 {
			sb.WriteString(PrintTree(n.Children, indent+1))
		}
	}
	return sb.String()
}
