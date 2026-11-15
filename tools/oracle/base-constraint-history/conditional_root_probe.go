// Research overlay only. No production checker/shim/provider API changes.
// Internal signatures read at TypeScript 3e9236361fa97a9e6c685f4833ac457ae2d8c50a.
package checker

import (
	"fmt"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
)

type ConditionalRootConstraintProbe struct {
	ShapeSurvived             bool
	InitiallyUncached         bool
	SeedCount                 int
	SeedsDistinctAndUnrelated bool
	SharedRecursionIdentity   bool
	ConstraintStatus          string
	NormalQueryStatus         string
	NegativeCached            bool
}

// ProbeConditionalRootConstraint exposes one fixed experiment, not general
// access to checker internals. Each call requires its own fresh checker.
func (c *Checker) ProbeConditionalRootConstraint(indexedNode, payloadNode *ast.Node, depth int) (ConditionalRootConstraintProbe, error) {
	p := ConditionalRootConstraintProbe{}
	if depth != 0 && depth != 9 {
		return p, fmt.Errorf("unsupported research depth %d", depth)
	}
	indexed := c.getTypeFromTypeNode(indexedNode)
	if indexed.flags&TypeFlagsIndexedAccess == 0 {
		return p, fmt.Errorf("INCONCLUSIVE/rejectcandidate: fixture is not a deferred indexed access")
	}
	index := indexed.AsIndexedAccessType().indexType
	if index.flags&TypeFlagsStringLiteral == 0 || index.AsLiteralType().Value() != "types" {
		return p, fmt.Errorf("INCONCLUSIVE/rejectcandidate: deferred index is not the captured types key")
	}
	object := indexed.AsIndexedAccessType().objectType
	if object.flags&TypeFlagsConditional == 0 || getRecursionIdentityTarget(indexed) != object {
		return p, fmt.Errorf("INCONCLUSIVE/rejectcandidate: indexed root is not the deferred conditional")
	}
	conditional := object.AsConditionalType()
	uncached := func() bool {
		return indexed.AsConstrainedType().resolvedBaseConstraint == nil &&
			object.AsConstrainedType().resolvedBaseConstraint == nil &&
			conditional.resolvedDefaultConstraint == nil &&
			conditional.resolvedConstraintOfDistributive == nil
	}
	if !uncached() {
		return p, fmt.Errorf("INCONCLUSIVE/rejectcandidate: type construction already warmed constraint caches")
	}
	simplified := c.getSimplifiedType(indexed, false)
	if simplified != indexed || simplified.flags&TypeFlagsIndexedAccess == 0 ||
		simplified.AsIndexedAccessType().objectType != object ||
		c.getSimplifiedType(object, false) != object {
		return p, fmt.Errorf("INCONCLUSIVE/rejectcandidate: simplification eliminated or changed the indexed/conditional shape")
	}
	p.ShapeSurvived = true
	p.InitiallyUncached = uncached()
	if !p.InitiallyUncached {
		return p, fmt.Errorf("INCONCLUSIVE/rejectcandidate: simplification warmed constraint caches")
	}
	p.SharedRecursionIdentity = getRecursionIdentity(indexed) == getRecursionIdentity(object)

	stack := make([]RecursionId, 0, depth)
	seen := map[RecursionId]bool{}
	for range depth {
		// Newly allocated, symbol-less parameters cannot belong to the fixture.
		identity := getRecursionIdentity(c.newTypeParameter(nil))
		if seen[identity] || identity == getRecursionIdentity(indexed) ||
			identity == getRecursionIdentity(object) || identity == asRecursionId(object) {
			return p, fmt.Errorf("invalid research seed: duplicate or related recursion identity")
		}
		seen[identity] = true
		stack = append(stack, identity)
	}
	p.SeedCount = len(stack)
	p.SeedsDistinctAndUnrelated = true

	// Do not warm the constraint via a cold query in the depth-nine history.
	constraint := c.getResolvedBaseConstraint(indexed, stack)
	normal := c.getBaseConstraintOfType(indexed)
	// Snapshot the tested caches before identity comparison or type printing can
	// perform additional checker work.
	for _, t := range []*Type{indexed, object} {
		cached := t.AsConstrainedType().resolvedBaseConstraint
		if cached == c.noConstraintType || cached == c.circularConstraintType {
			p.NegativeCached = true
		}
	}
	expected := c.getTypeFromTypeNode(payloadNode)
	status := func(t *Type) string {
		switch t {
		case nil:
			return "absent"
		case c.noConstraintType:
			return "noConstraintType"
		case c.circularConstraintType:
			return "circularConstraintType"
		default:
			if c.isTypeIdenticalTo(t, expected) {
				return "payload"
			}
			return "other: " + c.TypeToString(t)
		}
	}
	p.ConstraintStatus = status(constraint)
	p.NormalQueryStatus = status(normal)
	return p, nil
}
