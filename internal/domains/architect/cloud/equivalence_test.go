package cloud

import (
	"testing"
)

func TestMapServices(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	RegisterAWSServices(kb)
	RegisterGCPServices(kb)
	RegisterAzureServices(kb)
	RegisterAlibabaServices(kb)

	mappings := kb.MapServices(AWS, GCP)
	if len(mappings) == 0 {
		t.Fatal("expected at least one mapping from AWS to GCP")
	}

	// Verify mapping structure
	for _, m := range mappings {
		if m.SourceProvider != AWS {
			t.Errorf("expected source provider AWS, got %s", m.SourceProvider)
		}
		if m.TargetProvider != GCP {
			t.Errorf("expected target provider GCP, got %s", m.TargetProvider)
		}
		if m.SourceService == "" {
			t.Error("expected non-empty source service")
		}
		if m.Compatibility == "" {
			t.Error("expected non-empty compatibility")
		}
		validCompat := map[string]bool{
			"direct": true, "partial": true, "alternative": true, "none": true,
		}
		if !validCompat[m.Compatibility] {
			t.Errorf("invalid compatibility: %s", m.Compatibility)
		}
	}
}

func TestMapServicesBidirectional(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	RegisterAWSServices(kb)
	RegisterGCPServices(kb)

	awsToGCP := kb.MapServices(AWS, GCP)
	gcpToAWS := kb.MapServices(GCP, AWS)

	// Both directions should produce mappings
	if len(awsToGCP) == 0 {
		t.Fatal("expected AWS->GCP mappings")
	}
	if len(gcpToAWS) == 0 {
		t.Fatal("expected GCP->AWS mappings")
	}

	// Check that a known mapping exists in both directions
	foundS3toGCS := false
	for _, m := range awsToGCP {
		if m.SourceService == "S3" && m.TargetService == "Cloud Storage" {
			foundS3toGCS = true
			break
		}
	}
	if !foundS3toGCS {
		t.Error("expected S3 -> Cloud Storage mapping")
	}

	foundGCStoS3 := false
	for _, m := range gcpToAWS {
		if m.SourceService == "Cloud Storage" && m.TargetService == "S3" {
			foundGCStoS3 = true
			break
		}
	}
	if !foundGCStoS3 {
		t.Error("expected Cloud Storage -> S3 mapping")
	}
}

func TestMapServicesNoMappings(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	// Empty KB should return no mappings
	mappings := kb.MapServices(AWS, GCP)
	if len(mappings) != 0 {
		t.Fatalf("expected 0 mappings from empty KB, got %d", len(mappings))
	}
}

func TestServiceMappingCompatibility(t *testing.T) {
	kb := NewServiceKnowledgeBase()
	RegisterAWSServices(kb)
	RegisterGCPServices(kb)

	mappings := kb.MapServices(AWS, GCP)

	// At least some should be "direct" compatibility
	directCount := 0
	for _, m := range mappings {
		if m.Compatibility == "direct" {
			directCount++
		}
	}
	if directCount == 0 {
		t.Error("expected at least some direct compatibility mappings")
	}
}
