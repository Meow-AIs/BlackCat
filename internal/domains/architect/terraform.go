package architect

// TFModulePattern represents a Terraform/OpenTofu module pattern with best practices.
type TFModulePattern struct {
	Name          string
	Description   string
	Provider      string // "aws", "gcp", "azure", "any"
	Category      string // "compute", "networking", "database", "security", "storage", "monitoring"
	Template      string // HCL template
	BestPractices []string
	AntiPatterns  []string
	Variables     []TFVariable
}

// TFVariable describes an input variable for a Terraform module.
type TFVariable struct {
	Name        string
	Type        string // "string", "number", "bool", "list", "map"
	Description string
	Default     string
	Required    bool
}

// TFStateAdvice provides guidance on state backend configuration.
type TFStateAdvice struct {
	Backend      string // "s3", "gcs", "azurerm", "local"
	LockingTable string
	Encryption   bool
	BestPractice string
}

// OpenTofuInfo provides compatibility details for OpenTofu.
type OpenTofuInfo struct {
	Compatible       bool
	Notes            string
	ProviderRegistry string // "registry.opentofu.org"
	LicenseNote      string // "MPL-2.0 (fully open source)"
}

// LoadTerraformPatterns returns 15+ built-in Terraform module patterns.
func LoadTerraformPatterns() []TFModulePattern {
	return []TFModulePattern{
		{
			Name: "VPC Network", Description: "AWS VPC with public/private subnets", Provider: "aws", Category: "networking",
			Template:      "resource \"aws_vpc\" \"main\" {\n  cidr_block = var.cidr\n  enable_dns_hostnames = true\n}",
			BestPractices: []string{"Use private subnets for workloads", "Enable VPC flow logs", "Use NAT gateway for outbound"},
			AntiPatterns:  []string{"All resources in public subnets", "Single AZ deployment"},
			Variables: []TFVariable{
				{Name: "cidr", Type: "string", Description: "VPC CIDR block", Default: "10.0.0.0/16", Required: true},
				{Name: "azs", Type: "list", Description: "Availability zones", Required: true},
			},
		},
		{
			Name: "EKS Cluster", Description: "AWS EKS Kubernetes cluster with managed node groups", Provider: "aws", Category: "compute",
			Template:      "resource \"aws_eks_cluster\" \"main\" {\n  name     = var.cluster_name\n  role_arn = aws_iam_role.cluster.arn\n}",
			BestPractices: []string{"Use managed node groups", "Enable cluster logging", "Use private endpoint"},
			AntiPatterns:  []string{"Public API endpoint without restrictions", "Single node group for all workloads"},
			Variables: []TFVariable{
				{Name: "cluster_name", Type: "string", Description: "EKS cluster name", Required: true},
				{Name: "node_instance_type", Type: "string", Description: "EC2 instance type", Default: "t3.medium"},
			},
		},
		{
			Name: "RDS Database", Description: "AWS RDS with multi-AZ and encryption", Provider: "aws", Category: "database",
			Template:      "resource \"aws_db_instance\" \"main\" {\n  engine         = var.engine\n  instance_class = var.instance_class\n  multi_az       = true\n  storage_encrypted = true\n}",
			BestPractices: []string{"Enable multi-AZ", "Enable encryption at rest", "Use automated backups", "Set deletion protection"},
			AntiPatterns:  []string{"Public accessibility", "No backup retention", "Default credentials"},
			Variables: []TFVariable{
				{Name: "engine", Type: "string", Description: "Database engine", Default: "postgres", Required: true},
				{Name: "instance_class", Type: "string", Description: "RDS instance class", Default: "db.t3.medium"},
			},
		},
		{
			Name: "S3 Encrypted Bucket", Description: "S3 bucket with encryption, versioning, and access controls", Provider: "aws", Category: "storage",
			Template:      "resource \"aws_s3_bucket\" \"main\" {\n  bucket = var.bucket_name\n}\nresource \"aws_s3_bucket_server_side_encryption_configuration\" \"main\" {\n  bucket = aws_s3_bucket.main.id\n  rule { apply_server_side_encryption_by_default { sse_algorithm = \"aws:kms\" } }\n}",
			BestPractices: []string{"Enable versioning", "Enable server-side encryption", "Block public access", "Enable access logging"},
			AntiPatterns:  []string{"Public bucket policy", "No encryption", "No versioning"},
			Variables:     []TFVariable{{Name: "bucket_name", Type: "string", Description: "S3 bucket name", Required: true}},
		},
		{
			Name: "IAM Roles", Description: "IAM roles and policies with least privilege", Provider: "aws", Category: "security",
			Template:      "resource \"aws_iam_role\" \"main\" {\n  name               = var.role_name\n  assume_role_policy = data.aws_iam_policy_document.assume.json\n}",
			BestPractices: []string{"Use least privilege", "Use policy conditions", "Avoid wildcard permissions", "Use service-linked roles"},
			AntiPatterns:  []string{"Wildcard actions (*)", "Inline policies everywhere", "Root account usage"},
			Variables:     []TFVariable{{Name: "role_name", Type: "string", Description: "IAM role name", Required: true}},
		},
		{
			Name: "Lambda Function", Description: "AWS Lambda with VPC, logging, and X-Ray", Provider: "aws", Category: "compute",
			Template:      "resource \"aws_lambda_function\" \"main\" {\n  function_name = var.function_name\n  runtime       = var.runtime\n  handler       = var.handler\n  role          = aws_iam_role.lambda.arn\n}",
			BestPractices: []string{"Set memory and timeout appropriately", "Use environment variables for config", "Enable X-Ray tracing"},
			AntiPatterns:  []string{"Hardcoded secrets", "Oversized deployment packages", "No dead letter queue"},
			Variables: []TFVariable{
				{Name: "function_name", Type: "string", Description: "Lambda function name", Required: true},
				{Name: "runtime", Type: "string", Description: "Runtime", Default: "provided.al2023"},
			},
		},
		{
			Name: "API Gateway", Description: "REST or HTTP API Gateway with throttling", Provider: "aws", Category: "networking",
			Template:      "resource \"aws_apigatewayv2_api\" \"main\" {\n  name          = var.api_name\n  protocol_type = \"HTTP\"\n}",
			BestPractices: []string{"Enable access logging", "Configure throttling", "Use custom domain with TLS", "Enable WAF"},
			AntiPatterns:  []string{"No authentication", "No rate limiting", "Overly permissive CORS"},
			Variables:     []TFVariable{{Name: "api_name", Type: "string", Description: "API name", Required: true}},
		},
		{
			Name: "CloudFront CDN", Description: "CloudFront distribution with S3 origin", Provider: "aws", Category: "networking",
			Template:      "resource \"aws_cloudfront_distribution\" \"main\" {\n  origin {\n    domain_name = aws_s3_bucket.main.bucket_regional_domain_name\n  }\n  enabled = true\n  default_cache_behavior { viewer_protocol_policy = \"redirect-to-https\" }\n}",
			BestPractices: []string{"Enforce HTTPS", "Use OAI/OAC for S3", "Enable compression", "Set appropriate cache TTLs"},
			AntiPatterns:  []string{"HTTP allowed", "Direct S3 public access", "No custom error pages"},
			Variables:     []TFVariable{{Name: "domain_name", Type: "string", Description: "Custom domain", Required: false}},
		},
		{
			Name: "Route53 DNS", Description: "DNS zones and records with health checks", Provider: "aws", Category: "networking",
			Template:      "resource \"aws_route53_zone\" \"main\" {\n  name = var.domain\n}\nresource \"aws_route53_record\" \"main\" {\n  zone_id = aws_route53_zone.main.zone_id\n  name    = var.domain\n  type    = \"A\"\n}",
			BestPractices: []string{"Use alias records where possible", "Enable health checks", "Use routing policies for failover"},
			AntiPatterns:  []string{"No health checks", "Single region without failover"},
			Variables:     []TFVariable{{Name: "domain", Type: "string", Description: "Domain name", Required: true}},
		},
		{
			Name: "Security Group", Description: "Security group with minimal ingress rules", Provider: "aws", Category: "security",
			Template:      "resource \"aws_security_group\" \"main\" {\n  name   = var.name\n  vpc_id = var.vpc_id\n  ingress { from_port = 443; to_port = 443; protocol = \"tcp\"; cidr_blocks = var.allowed_cidrs }\n  egress { from_port = 0; to_port = 0; protocol = \"-1\"; cidr_blocks = [\"0.0.0.0/0\"] }\n}",
			BestPractices: []string{"Restrict ingress to specific CIDRs", "Use security group references over CIDRs", "Document each rule"},
			AntiPatterns:  []string{"0.0.0.0/0 ingress", "All ports open", "Overly broad egress"},
			Variables: []TFVariable{
				{Name: "name", Type: "string", Description: "Security group name", Required: true},
				{Name: "vpc_id", Type: "string", Description: "VPC ID", Required: true},
			},
		},
		{
			Name: "Auto Scaling Group", Description: "ASG with launch template and scaling policies", Provider: "aws", Category: "compute",
			Template:      "resource \"aws_autoscaling_group\" \"main\" {\n  min_size         = var.min_size\n  max_size         = var.max_size\n  desired_capacity = var.desired\n  launch_template { id = aws_launch_template.main.id }\n}",
			BestPractices: []string{"Use launch templates", "Configure health checks", "Use mixed instance policies", "Set appropriate cooldowns"},
			AntiPatterns:  []string{"Fixed desired count without scaling", "No health checks", "Single instance type"},
			Variables: []TFVariable{
				{Name: "min_size", Type: "number", Description: "Minimum instances", Default: "1", Required: true},
				{Name: "max_size", Type: "number", Description: "Maximum instances", Default: "3", Required: true},
			},
		},
		{
			Name: "ElastiCache Redis", Description: "ElastiCache Redis cluster with encryption", Provider: "aws", Category: "database",
			Template:      "resource \"aws_elasticache_replication_group\" \"main\" {\n  replication_group_id = var.cluster_name\n  engine               = \"redis\"\n  node_type            = var.node_type\n  at_rest_encryption_enabled = true\n  transit_encryption_enabled = true\n}",
			BestPractices: []string{"Enable encryption in transit and at rest", "Use multi-AZ", "Set eviction policies", "Monitor memory usage"},
			AntiPatterns:  []string{"No encryption", "Single node without replication", "No backup"},
			Variables:     []TFVariable{{Name: "cluster_name", Type: "string", Description: "Cluster name", Required: true}},
		},
		{
			Name: "SNS SQS Messaging", Description: "SNS topic with SQS subscriber and DLQ", Provider: "aws", Category: "compute",
			Template:      "resource \"aws_sns_topic\" \"main\" {\n  name = var.topic_name\n}\nresource \"aws_sqs_queue\" \"main\" {\n  name = var.queue_name\n  redrive_policy = jsonencode({ deadLetterTargetArn = aws_sqs_queue.dlq.arn, maxReceiveCount = 3 })\n}",
			BestPractices: []string{"Use dead letter queues", "Enable encryption", "Set visibility timeout appropriately", "Monitor queue depth"},
			AntiPatterns:  []string{"No DLQ", "Unencrypted messages", "No monitoring on queue depth"},
			Variables:     []TFVariable{{Name: "topic_name", Type: "string", Description: "SNS topic name", Required: true}},
		},
		{
			Name: "CloudWatch Monitoring", Description: "CloudWatch alarms, dashboards, and log groups", Provider: "aws", Category: "monitoring",
			Template:      "resource \"aws_cloudwatch_metric_alarm\" \"main\" {\n  alarm_name          = var.alarm_name\n  comparison_operator = \"GreaterThanThreshold\"\n  evaluation_periods  = 2\n  metric_name         = var.metric_name\n  namespace           = var.namespace\n  period              = 300\n  statistic           = \"Average\"\n  threshold           = var.threshold\n}",
			BestPractices: []string{"Set alarms on key metrics", "Use composite alarms", "Retain logs appropriately", "Use metric filters"},
			AntiPatterns:  []string{"No alarms", "Infinite log retention", "Alert fatigue from noisy alarms"},
			Variables:     []TFVariable{{Name: "alarm_name", Type: "string", Description: "Alarm name", Required: true}},
		},
		{
			Name: "KMS Encryption Key", Description: "KMS key with key policy and rotation", Provider: "aws", Category: "security",
			Template:      "resource \"aws_kms_key\" \"main\" {\n  description         = var.description\n  enable_key_rotation = true\n  deletion_window_in_days = 30\n}",
			BestPractices: []string{"Enable automatic key rotation", "Use key policies for access control", "Set deletion window", "Use separate keys per service"},
			AntiPatterns:  []string{"AWS managed keys for sensitive data", "No key rotation", "Overly permissive key policy"},
			Variables:     []TFVariable{{Name: "description", Type: "string", Description: "Key description", Required: true}},
		},
		{
			Name: "GKE Cluster", Description: "GCP GKE cluster with private nodes", Provider: "gcp", Category: "compute",
			Template:      "resource \"google_container_cluster\" \"main\" {\n  name     = var.cluster_name\n  location = var.region\n  private_cluster_config { enable_private_nodes = true }\n}",
			BestPractices: []string{"Use private nodes", "Enable workload identity", "Use release channels", "Enable network policy"},
			AntiPatterns:  []string{"Public nodes", "Default service account", "No network policy"},
			Variables:     []TFVariable{{Name: "cluster_name", Type: "string", Description: "GKE cluster name", Required: true}},
		},
	}
}

// GetTerraformPattern returns a pattern by exact name.
func GetTerraformPattern(name string) (TFModulePattern, bool) {
	for _, p := range LoadTerraformPatterns() {
		if p.Name == name {
			return p, true
		}
	}
	return TFModulePattern{}, false
}

// SearchTerraformPatterns finds patterns matching a query in name, description, or category.
func SearchTerraformPatterns(query string) []TFModulePattern {
	q := toLowerCase(query)
	var results []TFModulePattern
	for _, p := range LoadTerraformPatterns() {
		if containsLower(p.Name, q) || containsLower(p.Description, q) || containsLower(p.Category, q) {
			results = append(results, p)
		}
	}
	return results
}

// StateBackendAdvice returns state backend recommendations for a cloud provider.
func StateBackendAdvice(provider string) TFStateAdvice {
	switch toLowerCase(provider) {
	case "aws":
		return TFStateAdvice{
			Backend:      "s3",
			LockingTable: "terraform-locks",
			Encryption:   true,
			BestPractice: "Use S3 backend with DynamoDB locking and KMS encryption. Enable versioning on the state bucket.",
		}
	case "gcp":
		return TFStateAdvice{
			Backend:      "gcs",
			LockingTable: "built-in",
			Encryption:   true,
			BestPractice: "Use GCS backend with built-in locking. Enable object versioning and customer-managed encryption keys.",
		}
	case "azure":
		return TFStateAdvice{
			Backend:      "azurerm",
			LockingTable: "built-in",
			Encryption:   true,
			BestPractice: "Use Azure Storage Account backend with blob locking. Enable soft delete and customer-managed keys.",
		}
	default:
		return TFStateAdvice{
			Backend:      "local",
			LockingTable: "",
			Encryption:   false,
			BestPractice: "Local state is only for development. Migrate to a remote backend for team use.",
		}
	}
}

// GetOpenTofuInfo returns compatibility information for OpenTofu.
func GetOpenTofuInfo() OpenTofuInfo {
	return OpenTofuInfo{
		Compatible:       true,
		Notes:            "OpenTofu is a fork of Terraform 1.5.x. Most Terraform modules work unchanged. Some Terraform-specific features (cloud block, Sentinel) are not available.",
		ProviderRegistry: "registry.opentofu.org",
		LicenseNote:      "MPL-2.0 (fully open source)",
	}
}

// TerraformBestPractices returns 15+ general Terraform best practices.
func TerraformBestPractices() []string {
	return []string{
		"Use modules for reusable components",
		"Pin provider versions with version constraints",
		"Use remote state with locking for team collaboration",
		"Enable state encryption at rest",
		"Use workspaces for environments OR separate directory structure",
		"Run terraform fmt and terraform validate in CI",
		"Use tflint for linting HCL code",
		"Use checkov or tfsec for security scanning",
		"Use data sources instead of hardcoded values",
		"Tag all resources with owner, environment, and cost center",
		"Use count or for_each over copy-paste duplication",
		"Separate state files per environment",
		"Use -target sparingly and only for debugging",
		"Always run plan before apply and review the output",
		"Use import blocks (TF 1.5+) over terraform import command",
		"Store sensitive values in a secrets manager, not in tfvars",
		"Use moved blocks for refactoring without destroying resources",
	}
}
