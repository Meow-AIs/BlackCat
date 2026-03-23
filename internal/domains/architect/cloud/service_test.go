package cloud

import (
	"testing"
)

func TestNewServiceKnowledgeBase(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	if kb == nil {
		t.Fatal("expected non-nil knowledge base")
	}
	if kb.Count() != 0 {
		t.Fatalf("expected empty kb, got %d", kb.Count())
	}
}

func TestAddAndGet(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	svc := CloudService{
		Name:        "S3",
		Provider:    AWS,
		Category:    CatStorage,
		Description: "Object storage",
		UseCases:    []string{"static hosting", "data lake"},
		Equivalents: map[CloudProvider]string{GCP: "Cloud Storage", Azure: "Blob Storage"},
		Tags:        []string{"storage", "object"},
	}
	kb.Add(svc)

	got, ok := kb.Get(AWS, "S3")
	if !ok {
		t.Fatal("expected to find S3")
	}
	if got.Name != "S3" {
		t.Fatalf("expected S3, got %s", got.Name)
	}
	if got.Provider != AWS {
		t.Fatalf("expected AWS, got %s", got.Provider)
	}

	_, ok = kb.Get(AWS, "nonexistent")
	if ok {
		t.Fatal("expected not found for nonexistent service")
	}

	_, ok = kb.Get(GCP, "S3")
	if ok {
		t.Fatal("expected not found for wrong provider")
	}
}

func TestCount(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	kb.Add(CloudService{Name: "S3", Provider: AWS, Category: CatStorage})
	kb.Add(CloudService{Name: "EC2", Provider: AWS, Category: CatCompute})
	kb.Add(CloudService{Name: "Compute Engine", Provider: GCP, Category: CatCompute})

	if kb.Count() != 3 {
		t.Fatalf("expected 3, got %d", kb.Count())
	}
}

func TestByProvider(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	kb.Add(CloudService{Name: "S3", Provider: AWS, Category: CatStorage})
	kb.Add(CloudService{Name: "EC2", Provider: AWS, Category: CatCompute})
	kb.Add(CloudService{Name: "Compute Engine", Provider: GCP, Category: CatCompute})

	awsServices := kb.ByProvider(AWS)
	if len(awsServices) != 2 {
		t.Fatalf("expected 2 AWS services, got %d", len(awsServices))
	}

	gcpServices := kb.ByProvider(GCP)
	if len(gcpServices) != 1 {
		t.Fatalf("expected 1 GCP service, got %d", len(gcpServices))
	}

	aliServices := kb.ByProvider(Alibaba)
	if len(aliServices) != 0 {
		t.Fatalf("expected 0 Alibaba services, got %d", len(aliServices))
	}
}

func TestByCategory(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	kb.Add(CloudService{Name: "S3", Provider: AWS, Category: CatStorage})
	kb.Add(CloudService{Name: "Cloud Storage", Provider: GCP, Category: CatStorage})
	kb.Add(CloudService{Name: "EC2", Provider: AWS, Category: CatCompute})

	storageServices := kb.ByCategory(CatStorage)
	if len(storageServices) != 2 {
		t.Fatalf("expected 2 storage services, got %d", len(storageServices))
	}

	computeServices := kb.ByCategory(CatCompute)
	if len(computeServices) != 1 {
		t.Fatalf("expected 1 compute service, got %d", len(computeServices))
	}
}

func TestSearch(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	kb.Add(CloudService{
		Name:     "S3",
		Provider: AWS,
		Category: CatStorage,
		Tags:     []string{"object", "storage"},
	})
	kb.Add(CloudService{
		Name:     "Cloud Storage",
		Provider: GCP,
		Category: CatStorage,
		Tags:     []string{"object", "storage"},
	})
	kb.Add(CloudService{
		Name:     "EC2",
		Provider: AWS,
		Category: CatCompute,
		Tags:     []string{"vm", "compute"},
	})

	// Search by name substring
	results := kb.Search("S3")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'S3', got %d", len(results))
	}

	// Search by tag
	results = kb.Search("storage")
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'storage', got %d", len(results))
	}

	// Search by category name
	results = kb.Search("compute")
	if len(results) < 1 {
		t.Fatal("expected at least 1 result for 'compute'")
	}

	// Search case-insensitive
	results = kb.Search("STORAGE")
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'STORAGE', got %d", len(results))
	}

	// No results
	results = kb.Search("nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestFindEquivalent(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	kb.Add(CloudService{
		Name:        "S3",
		Provider:    AWS,
		Category:    CatStorage,
		Equivalents: map[CloudProvider]string{GCP: "Cloud Storage", Azure: "Blob Storage"},
	})
	kb.Add(CloudService{
		Name:        "Cloud Storage",
		Provider:    GCP,
		Category:    CatStorage,
		Equivalents: map[CloudProvider]string{AWS: "S3", Azure: "Blob Storage"},
	})

	// Find GCP equivalent of AWS S3
	equiv, ok := kb.FindEquivalent(AWS, "S3", GCP)
	if !ok {
		t.Fatal("expected to find GCP equivalent of S3")
	}
	if equiv.Name != "Cloud Storage" {
		t.Fatalf("expected Cloud Storage, got %s", equiv.Name)
	}

	// Find AWS equivalent of GCP Cloud Storage
	equiv, ok = kb.FindEquivalent(GCP, "Cloud Storage", AWS)
	if !ok {
		t.Fatal("expected to find AWS equivalent of Cloud Storage")
	}
	if equiv.Name != "S3" {
		t.Fatalf("expected S3, got %s", equiv.Name)
	}

	// Not found: wrong source
	_, ok = kb.FindEquivalent(AWS, "nonexistent", GCP)
	if ok {
		t.Fatal("expected not found for nonexistent service")
	}

	// Not found: no mapping to target
	_, ok = kb.FindEquivalent(AWS, "S3", Alibaba)
	if ok {
		t.Fatal("expected not found for unmapped target provider")
	}
}

func TestBuiltinServicesLoaded(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	RegisterAWSServices(kb)
	RegisterGCPServices(kb)
	RegisterAzureServices(kb)
	RegisterAlibabaServices(kb)

	// Check minimum counts
	awsCount := len(kb.ByProvider(AWS))
	if awsCount < 30 {
		t.Fatalf("expected at least 30 AWS services, got %d", awsCount)
	}

	gcpCount := len(kb.ByProvider(GCP))
	if gcpCount < 25 {
		t.Fatalf("expected at least 25 GCP services, got %d", gcpCount)
	}

	azureCount := len(kb.ByProvider(Azure))
	if azureCount < 25 {
		t.Fatalf("expected at least 25 Azure services, got %d", azureCount)
	}

	aliCount := len(kb.ByProvider(Alibaba))
	if aliCount < 15 {
		t.Fatalf("expected at least 15 Alibaba services, got %d", aliCount)
	}

	// Verify specific services exist
	tests := []struct {
		provider CloudProvider
		name     string
	}{
		{AWS, "EC2"},
		{AWS, "Lambda"},
		{AWS, "S3"},
		{AWS, "DynamoDB"},
		{AWS, "SageMaker"},
		{GCP, "Compute Engine"},
		{GCP, "BigQuery"},
		{GCP, "Cloud Run"},
		{Azure, "Virtual Machines"},
		{Azure, "Cosmos DB"},
		{Azure, "AKS"},
		{Alibaba, "ECS"},
		{Alibaba, "OSS"},
		{Alibaba, "PolarDB"},
	}

	for _, tt := range tests {
		if _, ok := kb.Get(tt.provider, tt.name); !ok {
			t.Errorf("expected %s %s to exist", tt.provider, tt.name)
		}
	}
}

func TestAllCategoriesCovered(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	RegisterAWSServices(kb)

	categories := []ServiceCategory{
		CatCompute, CatStorage, CatDatabase, CatNetworking, CatSecurity,
		CatAI_ML, CatContainers, CatServerless, CatMessaging, CatAnalytics,
		CatMonitoring, CatDevTools, CatIdentity,
	}

	for _, cat := range categories {
		services := kb.ByCategory(cat)
		if len(services) == 0 {
			t.Errorf("expected at least 1 service in category %s", cat)
		}
	}
}
