package cloud

// ServiceMapping represents a cross-cloud service equivalence.
type ServiceMapping struct {
	SourceService  string
	TargetService  string
	SourceProvider CloudProvider
	TargetProvider CloudProvider
	Compatibility  string // "direct", "partial", "alternative", "none"
	Notes          string
}

// MapServices produces all known mappings from one provider to another.
func (kb *ServiceKnowledgeBase) MapServices(from, to CloudProvider) []ServiceMapping {
	var mappings []ServiceMapping
	for _, svc := range kb.services {
		if svc.Provider != from {
			continue
		}
		targetName, ok := svc.Equivalents[to]
		if !ok {
			continue
		}
		compat := classifyCompatibility(svc, targetName, to, kb)
		mappings = append(mappings, ServiceMapping{
			SourceService:  svc.Name,
			TargetService:  targetName,
			SourceProvider: from,
			TargetProvider: to,
			Compatibility:  compat,
			Notes:          compatNotes(svc.Category, compat),
		})
	}
	return mappings
}

// classifyCompatibility determines how closely two services match.
func classifyCompatibility(
	source CloudService, targetName string, targetProvider CloudProvider, kb *ServiceKnowledgeBase,
) string {
	target, ok := kb.Get(targetProvider, targetName)
	if !ok {
		return "none"
	}
	// Same category = direct, different category = alternative
	if target.Category == source.Category {
		// Check if the target also maps back to source (bidirectional = direct)
		if backName, ok := target.Equivalents[source.Provider]; ok && backName == source.Name {
			return "direct"
		}
		return "partial"
	}
	return "alternative"
}

// compatNotes generates human-readable notes about the mapping.
func compatNotes(cat ServiceCategory, compat string) string {
	switch compat {
	case "direct":
		return "Functionally equivalent service in the " + string(cat) + " category"
	case "partial":
		return "Similar service with some feature differences"
	case "alternative":
		return "Alternative approach; not a 1:1 mapping"
	default:
		return "No equivalent service found"
	}
}
