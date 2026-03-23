package architect

import "testing"

func TestRecommendDatabase_Relational(t *testing.T) {
	req := DBRequirements{
		DataModel:        "relational",
		ScaleNeeds:       "medium",
		ConsistencyNeed:  "strong",
		NeedTransactions: true,
		Budget:           "low",
	}
	recs := RecommendDatabase(req)
	if len(recs) < 3 {
		t.Fatalf("expected at least 3 recommendations, got %d", len(recs))
	}
	// PostgreSQL should rank high for relational + strong consistency + transactions
	if recs[0].Category != DBRelational {
		t.Errorf("expected relational DB first, got %q (%s)", recs[0].Name, recs[0].Category)
	}
	if recs[0].Score <= 0 || recs[0].Score > 100 {
		t.Errorf("score out of range: %f", recs[0].Score)
	}
	if recs[0].Reason == "" {
		t.Error("expected a reason")
	}
}

func TestRecommendDatabase_Document(t *testing.T) {
	req := DBRequirements{
		DataModel:       "document",
		ScaleNeeds:      "large",
		ConsistencyNeed: "eventual",
		ReadHeavy:       true,
	}
	recs := RecommendDatabase(req)
	if len(recs) < 3 {
		t.Fatalf("expected at least 3 recommendations, got %d", len(recs))
	}
	found := false
	for _, r := range recs[:3] {
		if r.Name == "MongoDB" || r.Name == "DynamoDB" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MongoDB or DynamoDB in top 3 for document model")
	}
}

func TestRecommendDatabase_Graph(t *testing.T) {
	req := DBRequirements{
		DataModel: "graph",
	}
	recs := RecommendDatabase(req)
	found := false
	for _, r := range recs[:2] {
		if r.Name == "Neo4j" {
			found = true
		}
	}
	if !found {
		t.Error("expected Neo4j in top 2 for graph model")
	}
}

func TestRecommendDatabase_TimeSeries(t *testing.T) {
	req := DBRequirements{
		DataModel:  "time-series",
		ScaleNeeds: "large",
		WriteHeavy: true,
	}
	recs := RecommendDatabase(req)
	found := false
	for _, r := range recs[:3] {
		if r.Name == "InfluxDB" || r.Name == "TimescaleDB" || r.Name == "ClickHouse" {
			found = true
		}
	}
	if !found {
		t.Error("expected a time-series DB in top 3")
	}
}

func TestRecommendDatabase_Vector(t *testing.T) {
	req := DBRequirements{
		DataModel: "vector",
	}
	recs := RecommendDatabase(req)
	found := false
	for _, r := range recs[:3] {
		if r.Name == "Pinecone" || r.Name == "Qdrant" {
			found = true
		}
	}
	if !found {
		t.Error("expected a vector DB in top 3")
	}
}

func TestRecommendDatabase_KeyValue(t *testing.T) {
	req := DBRequirements{
		DataModel:  "key-value",
		ReadHeavy:  true,
		ScaleNeeds: "small",
		Budget:     "free",
	}
	recs := RecommendDatabase(req)
	found := false
	for _, r := range recs[:3] {
		if r.Name == "Redis" {
			found = true
		}
	}
	if !found {
		t.Error("expected Redis in top 3 for key-value")
	}
}

func TestRecommendDatabase_SortedByScore(t *testing.T) {
	req := DBRequirements{
		DataModel:       "relational",
		ConsistencyNeed: "strong",
	}
	recs := RecommendDatabase(req)
	for i := 1; i < len(recs); i++ {
		if recs[i].Score > recs[i-1].Score {
			t.Errorf("results not sorted: [%d]=%f > [%d]=%f",
				i, recs[i].Score, i-1, recs[i-1].Score)
		}
	}
}

func TestRecommendDatabase_StrengthsAndWeaknesses(t *testing.T) {
	req := DBRequirements{DataModel: "relational"}
	recs := RecommendDatabase(req)
	for _, r := range recs {
		if len(r.Strengths) == 0 {
			t.Errorf("%s missing strengths", r.Name)
		}
		if len(r.Weaknesses) == 0 {
			t.Errorf("%s missing weaknesses", r.Name)
		}
	}
}

func TestRecommendDatabase_CloudProvider(t *testing.T) {
	req := DBRequirements{
		DataModel:     "relational",
		CloudProvider: "aws",
	}
	recs := RecommendDatabase(req)
	// AWS-native options should get a boost
	if len(recs) < 3 {
		t.Fatalf("expected at least 3, got %d", len(recs))
	}
}

func TestRecommendDatabase_SmallScaleFree(t *testing.T) {
	req := DBRequirements{
		DataModel:  "relational",
		ScaleNeeds: "small",
		Budget:     "free",
	}
	recs := RecommendDatabase(req)
	// SQLite should appear for small + free
	found := false
	for _, r := range recs[:3] {
		if r.Name == "SQLite" {
			found = true
		}
	}
	if !found {
		t.Error("expected SQLite in top 3 for small + free relational")
	}
}
