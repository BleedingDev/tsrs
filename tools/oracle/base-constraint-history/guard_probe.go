// Included only by the research Go overlay, never by the compiler source patch.
package checker

import "slices"

type BaseConstraintGuardProbe struct {
	IndexedParameterVisits int
	DeepIndexedVisits      int
	RepeatedIdentityCuts   int
	MaxStackDepth          int
}

var baseConstraintGuardProbe BaseConstraintGuardProbe

func ResetBaseConstraintGuardProbe() {
	baseConstraintGuardProbe = BaseConstraintGuardProbe{}
}

func ReadBaseConstraintGuardProbe() BaseConstraintGuardProbe {
	return baseConstraintGuardProbe
}

func recordBaseConstraintGuard(t *Type, stack []RecursionId, identity RecursionId) {
	if t.flags&TypeFlagsIndexedAccess == 0 {
		return
	}
	target := t.AsIndexedAccessType().objectType
	for target.flags&TypeFlagsIndexedAccess != 0 {
		target = target.AsIndexedAccessType().objectType
	}
	if target.flags&TypeFlagsTypeParameter == 0 {
		return
	}
	p := &baseConstraintGuardProbe
	p.IndexedParameterVisits++
	p.MaxStackDepth = max(p.MaxStackDepth, len(stack))
	if len(stack) >= 10 {
		p.DeepIndexedVisits++
		if slices.Contains(stack, identity) {
			p.RepeatedIdentityCuts++
		}
	}
}
