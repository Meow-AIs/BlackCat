package cloud

import "strings"

// CloudProvider represents a major cloud platform.
type CloudProvider string

const (
	AWS     CloudProvider = "aws"
	GCP     CloudProvider = "gcp"
	Azure   CloudProvider = "azure"
	Alibaba CloudProvider = "alibaba"
)

// ServiceCategory classifies cloud services by function.
type ServiceCategory string

const (
	CatCompute    ServiceCategory = "compute"
	CatStorage    ServiceCategory = "storage"
	CatDatabase   ServiceCategory = "database"
	CatNetworking ServiceCategory = "networking"
	CatSecurity   ServiceCategory = "security"
	CatAI_ML      ServiceCategory = "ai-ml"
	CatContainers ServiceCategory = "containers"
	CatServerless ServiceCategory = "serverless"
	CatMessaging  ServiceCategory = "messaging"
	CatAnalytics  ServiceCategory = "analytics"
	CatMonitoring ServiceCategory = "monitoring"
	CatDevTools   ServiceCategory = "devtools"
	CatIdentity   ServiceCategory = "identity"
)

// ServiceTier indicates the cost tier of a service.
type ServiceTier string

const (
	TierFree       ServiceTier = "free"
	TierLow        ServiceTier = "low"
	TierMedium     ServiceTier = "medium"
	TierHigh       ServiceTier = "high"
	TierEnterprise ServiceTier = "enterprise"
)

// CloudService describes a cloud provider service with metadata.
type CloudService struct {
	Name         string
	Provider     CloudProvider
	Category     ServiceCategory
	Description  string
	UseCases     []string
	AntiPatterns []string
	PricingModel string
	FreeTier     string
	Equivalents  map[CloudProvider]string
	Tags         []string
}

// serviceKey creates a unique key for provider+name lookup.
func serviceKey(provider CloudProvider, name string) string {
	return string(provider) + "::" + name
}

// ServiceKnowledgeBase stores and queries cloud services.
type ServiceKnowledgeBase struct {
	services map[string]CloudService
}

// NewServiceKnowledgeBase creates an empty knowledge base.
func NewServiceKnowledgeBase() *ServiceKnowledgeBase {
	return &ServiceKnowledgeBase{services: make(map[string]CloudService)}
}

// Add registers a cloud service.
func (kb *ServiceKnowledgeBase) Add(svc CloudService) {
	kb.services[serviceKey(svc.Provider, svc.Name)] = svc
}

// Get returns a service by provider and name.
func (kb *ServiceKnowledgeBase) Get(provider CloudProvider, name string) (CloudService, bool) {
	svc, ok := kb.services[serviceKey(provider, name)]
	return svc, ok
}

// Search finds services matching a query against name, category, and tags.
func (kb *ServiceKnowledgeBase) Search(query string) []CloudService {
	q := strings.ToLower(query)
	var results []CloudService
	for _, svc := range kb.services {
		if matchesQuery(svc, q) {
			results = append(results, svc)
		}
	}
	return results
}

// matchesQuery checks if a service matches a lowercase query string.
func matchesQuery(svc CloudService, q string) bool {
	if strings.Contains(strings.ToLower(svc.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(string(svc.Category)), q) {
		return true
	}
	for _, tag := range svc.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}

// ByProvider returns all services for a given provider.
func (kb *ServiceKnowledgeBase) ByProvider(provider CloudProvider) []CloudService {
	var results []CloudService
	for _, svc := range kb.services {
		if svc.Provider == provider {
			results = append(results, svc)
		}
	}
	return results
}

// ByCategory returns all services in a given category.
func (kb *ServiceKnowledgeBase) ByCategory(cat ServiceCategory) []CloudService {
	var results []CloudService
	for _, svc := range kb.services {
		if svc.Category == cat {
			results = append(results, svc)
		}
	}
	return results
}

// FindEquivalent finds the equivalent service on a target provider.
func (kb *ServiceKnowledgeBase) FindEquivalent(
	provider CloudProvider, name string, targetProvider CloudProvider,
) (CloudService, bool) {
	source, ok := kb.Get(provider, name)
	if !ok {
		return CloudService{}, false
	}
	targetName, ok := source.Equivalents[targetProvider]
	if !ok {
		return CloudService{}, false
	}
	return kb.Get(targetProvider, targetName)
}

// Count returns the total number of services in the knowledge base.
func (kb *ServiceKnowledgeBase) Count() int {
	return len(kb.services)
}
