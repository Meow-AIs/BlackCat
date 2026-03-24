package architect

import "sort"

// DBCategory classifies the type of database.
type DBCategory string

const (
	DBRelational DBCategory = "relational"
	DBDocument   DBCategory = "document"
	DBKeyValue   DBCategory = "key-value"
	DBGraph      DBCategory = "graph"
	DBTimeSeries DBCategory = "time-series"
	DBVector     DBCategory = "vector"
	DBWideColumn DBCategory = "wide-column"
)

// DBRecommendation holds a scored database recommendation.
type DBRecommendation struct {
	Name       string
	Category   DBCategory
	Score      float64
	Reason     string
	Strengths  []string
	Weaknesses []string
}

// DBRequirements describes the requirements for database selection.
type DBRequirements struct {
	DataModel        string
	ScaleNeeds       string
	ConsistencyNeed  string
	ReadHeavy        bool
	WriteHeavy       bool
	NeedTransactions bool
	NeedFullText     bool
	NeedGeospatial   bool
	CloudProvider    string
	Budget           string
}

type dbProfile struct {
	Name       string
	Category   DBCategory
	Strengths  []string
	Weaknesses []string
	// scoring traits
	Models       []string // data models it supports well
	ScaleTiers   []string // scale tiers it handles
	Consistency  []string // consistency models
	ReadOpt      bool
	WriteOpt     bool
	Transactions bool
	FullText     bool
	Geospatial   bool
	Cloud        []string // cloud providers with managed offerings
	FreeTier     bool
	Reason       string
}

func allDatabases() []dbProfile {
	return []dbProfile{
		{
			Name: "PostgreSQL", Category: DBRelational,
			Strengths:   []string{"ACID compliance", "extensible", "rich SQL", "JSON support", "strong ecosystem"},
			Weaknesses:  []string{"horizontal scaling complexity", "write-heavy at massive scale"},
			Models:      []string{"relational", "document", "key-value"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"strong"}, ReadOpt: true, WriteOpt: false,
			Transactions: true, FullText: true, Geospatial: true,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Battle-tested relational DB with excellent extensibility",
		},
		{
			Name: "MySQL", Category: DBRelational,
			Strengths:   []string{"widespread adoption", "simple setup", "good read performance"},
			Weaknesses:  []string{"limited JSON support", "less extensible than PostgreSQL"},
			Models:      []string{"relational"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"strong"}, ReadOpt: true, WriteOpt: false,
			Transactions: true, FullText: true, Geospatial: true,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Popular relational DB with wide hosting support",
		},
		{
			Name: "SQLite", Category: DBRelational,
			Strengths:   []string{"zero setup", "embedded", "single file", "no server needed"},
			Weaknesses:  []string{"single writer", "no network access", "limited concurrency"},
			Models:      []string{"relational"},
			ScaleTiers:  []string{"small"},
			Consistency: []string{"strong"}, ReadOpt: true, WriteOpt: false,
			Transactions: true, FullText: true, Geospatial: false,
			Cloud: []string{}, FreeTier: true,
			Reason: "Perfect embedded DB for local/small-scale use",
		},
		{
			Name: "MongoDB", Category: DBDocument,
			Strengths:   []string{"flexible schema", "horizontal scaling", "rich query language"},
			Weaknesses:  []string{"memory hungry", "eventual consistency by default"},
			Models:      []string{"document", "key-value"},
			ScaleTiers:  []string{"small", "medium", "large", "massive"},
			Consistency: []string{"eventual", "tunable", "strong"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: true, FullText: true, Geospatial: true,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Leading document DB with flexible schema and scaling",
		},
		{
			Name: "DynamoDB", Category: DBDocument,
			Strengths:   []string{"fully managed", "auto-scaling", "single-digit ms latency"},
			Weaknesses:  []string{"AWS lock-in", "complex pricing", "limited query flexibility"},
			Models:      []string{"document", "key-value"},
			ScaleTiers:  []string{"small", "medium", "large", "massive"},
			Consistency: []string{"eventual", "strong"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: true, FullText: false, Geospatial: false,
			Cloud: []string{"aws"}, FreeTier: true,
			Reason: "AWS-native serverless document/KV store",
		},
		{
			Name: "Redis", Category: DBKeyValue,
			Strengths:   []string{"sub-millisecond latency", "rich data structures", "pub/sub"},
			Weaknesses:  []string{"memory-bound", "persistence trade-offs"},
			Models:      []string{"key-value", "document"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"eventual", "strong"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: false, FullText: true, Geospatial: true,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Ultra-fast in-memory key-value store",
		},
		{
			Name: "Cassandra", Category: DBWideColumn,
			Strengths:   []string{"linear scalability", "high write throughput", "multi-datacenter"},
			Weaknesses:  []string{"complex operations", "eventual consistency", "no joins"},
			Models:      []string{"wide-column", "key-value"},
			ScaleTiers:  []string{"large", "massive"},
			Consistency: []string{"eventual", "tunable"},
			ReadOpt:     false, WriteOpt: true,
			Transactions: false, FullText: false, Geospatial: false,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: false,
			Reason: "Distributed wide-column store for massive write workloads",
		},
		{
			Name: "Neo4j", Category: DBGraph,
			Strengths:   []string{"native graph storage", "Cypher query language", "relationship traversal"},
			Weaknesses:  []string{"scaling limitations", "niche use case", "memory intensive"},
			Models:      []string{"graph"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"strong"},
			ReadOpt:     true, WriteOpt: false,
			Transactions: true, FullText: true, Geospatial: true,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Leading graph database for relationship-heavy data",
		},
		{
			Name: "InfluxDB", Category: DBTimeSeries,
			Strengths:   []string{"purpose-built for time-series", "fast ingestion", "downsampling"},
			Weaknesses:  []string{"limited query flexibility", "clustering is commercial"},
			Models:      []string{"time-series"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"eventual"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: false, FullText: false, Geospatial: false,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Purpose-built time-series database with fast ingestion",
		},
		{
			Name: "TimescaleDB", Category: DBTimeSeries,
			Strengths:   []string{"PostgreSQL compatible", "automatic partitioning", "SQL support"},
			Weaknesses:  []string{"PostgreSQL overhead", "scaling complexity"},
			Models:      []string{"time-series", "relational"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"strong"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: true, FullText: true, Geospatial: true,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Time-series on PostgreSQL with full SQL support",
		},
		{
			Name: "Pinecone", Category: DBVector,
			Strengths:   []string{"fully managed", "fast similarity search", "simple API"},
			Weaknesses:  []string{"vendor lock-in", "cost at scale", "limited filtering"},
			Models:      []string{"vector"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"eventual"},
			ReadOpt:     true, WriteOpt: false,
			Transactions: false, FullText: false, Geospatial: false,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Managed vector database for AI/ML similarity search",
		},
		{
			Name: "Qdrant", Category: DBVector,
			Strengths:   []string{"open source", "rich filtering", "Rust-based performance"},
			Weaknesses:  []string{"younger ecosystem", "smaller community"},
			Models:      []string{"vector"},
			ScaleTiers:  []string{"small", "medium", "large"},
			Consistency: []string{"eventual", "strong"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: false, FullText: false, Geospatial: false,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "High-performance open-source vector database",
		},
		{
			Name: "ScyllaDB", Category: DBWideColumn,
			Strengths:   []string{"Cassandra compatible", "C++ performance", "low latency"},
			Weaknesses:  []string{"smaller community", "complex operations"},
			Models:      []string{"wide-column", "key-value"},
			ScaleTiers:  []string{"large", "massive"},
			Consistency: []string{"eventual", "tunable"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: false, FullText: false, Geospatial: false,
			Cloud: []string{"aws", "gcp"}, FreeTier: false,
			Reason: "High-performance Cassandra-compatible wide-column store",
		},
		{
			Name: "CockroachDB", Category: DBRelational,
			Strengths:   []string{"distributed SQL", "strong consistency", "PostgreSQL compatible"},
			Weaknesses:  []string{"latency overhead", "complex pricing", "resource hungry"},
			Models:      []string{"relational"},
			ScaleTiers:  []string{"medium", "large", "massive"},
			Consistency: []string{"strong"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: true, FullText: false, Geospatial: true,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Distributed SQL with strong consistency and auto-scaling",
		},
		{
			Name: "ClickHouse", Category: DBTimeSeries,
			Strengths:   []string{"blazing analytics", "column-oriented", "high compression"},
			Weaknesses:  []string{"not for OLTP", "complex operations", "no updates"},
			Models:      []string{"time-series", "relational"},
			ScaleTiers:  []string{"medium", "large", "massive"},
			Consistency: []string{"eventual"},
			ReadOpt:     true, WriteOpt: true,
			Transactions: false, FullText: false, Geospatial: false,
			Cloud: []string{"aws", "gcp", "azure"}, FreeTier: true,
			Reason: "Column-oriented OLAP database for real-time analytics",
		},
	}
}

// RecommendDatabase scores and returns the top database recommendations.
func RecommendDatabase(req DBRequirements) []DBRecommendation {
	dbs := allDatabases()
	scored := make([]DBRecommendation, 0, len(dbs))

	for _, db := range dbs {
		score := scoreDatabase(db, req)
		scored = append(scored, DBRecommendation{
			Name:       db.Name,
			Category:   db.Category,
			Score:      score,
			Reason:     db.Reason,
			Strengths:  db.Strengths,
			Weaknesses: db.Weaknesses,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	top := 5
	if len(scored) < top {
		top = len(scored)
	}
	return scored[:top]
}

func scoreDatabase(db dbProfile, req DBRequirements) float64 {
	score := 20.0 // base score

	// Data model match (heaviest weight)
	if req.DataModel != "" {
		if sliceContains(db.Models, req.DataModel) {
			score += 35
			// Primary category bonus
			if string(db.Category) == req.DataModel {
				score += 10
			}
		}
	}

	// Scale match
	if req.ScaleNeeds != "" {
		if sliceContains(db.ScaleTiers, req.ScaleNeeds) {
			score += 10
		} else {
			score -= 15
		}
	}

	// Consistency match
	if req.ConsistencyNeed != "" {
		if sliceContains(db.Consistency, req.ConsistencyNeed) {
			score += 10
		} else {
			score -= 10
		}
	}

	// Read/write optimization
	if req.ReadHeavy && db.ReadOpt {
		score += 5
	}
	if req.WriteHeavy && db.WriteOpt {
		score += 5
	}

	// Feature requirements
	if req.NeedTransactions {
		if db.Transactions {
			score += 8
		} else {
			score -= 10
		}
	}
	if req.NeedFullText {
		if db.FullText {
			score += 5
		} else {
			score -= 5
		}
	}
	if req.NeedGeospatial {
		if db.Geospatial {
			score += 5
		} else {
			score -= 5
		}
	}

	// Cloud provider match
	if req.CloudProvider != "" && req.CloudProvider != "any" {
		if sliceContains(db.Cloud, req.CloudProvider) {
			score += 5
		} else {
			score -= 5
		}
	}

	// Budget
	if req.Budget == "free" {
		if db.FreeTier {
			score += 5
		} else {
			score -= 10
		}
	}

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
