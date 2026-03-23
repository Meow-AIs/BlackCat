package pipeline

import (
	"fmt"
	"strings"
)

// Generate dispatches to a platform-specific generator and returns the result.
func Generate(req PipelineRequest) (PipelineResult, error) {
	if req.ProjectName == "" {
		return PipelineResult{}, fmt.Errorf("project name is required")
	}
	if !supportedLanguages[req.Language] {
		return PipelineResult{}, fmt.Errorf("unsupported language: %q", req.Language)
	}

	switch req.Platform {
	case PlatformGitHubActions:
		return generateGitHubActions(req)
	case PlatformGitLabCI:
		return generateGitLabCI(req)
	default:
		return PipelineResult{}, fmt.Errorf("unsupported platform: %q", req.Platform)
	}
}

// ---------------------------------------------------------------------------
// GitHub Actions
// ---------------------------------------------------------------------------

func generateGitHubActions(req PipelineRequest) (PipelineResult, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("name: %s CI\n\n", req.ProjectName))
	b.WriteString("on:\n  push:\n    branches: [main]\n  pull_request:\n    branches: [main]\n\n")
	b.WriteString("permissions: read-all\n\n")
	b.WriteString("jobs:\n")

	// Build & test job
	b.WriteString("  build:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")

	writeGHALanguageSteps(&b, req)

	if req.IncludeSecurityGates {
		writeGHASecuritySteps(&b)
	}

	if req.IncludeDocker {
		writeGHADockerSteps(&b, req)
	}

	return PipelineResult{
		Platform: PlatformGitHubActions,
		Filename: ".github/workflows/ci.yml",
		Content:  b.String(),
	}, nil
}

func writeGHALanguageSteps(b *strings.Builder, req PipelineRequest) {
	switch req.Language {
	case LangGo:
		ver := defaultIfEmpty(req.GoVersion, "1.22")
		b.WriteString(fmt.Sprintf("      - uses: actions/setup-go@v5\n        with:\n          go-version: '%s'\n", ver))
		b.WriteString("      - run: go mod download\n")
		b.WriteString("      - run: go vet ./...\n")
		b.WriteString("      - run: go test -race -coverprofile=coverage.out ./...\n")
		b.WriteString("      - run: go build -o bin/ ./...\n")

	case LangNode:
		ver := defaultIfEmpty(req.NodeVersion, "20")
		b.WriteString(fmt.Sprintf("      - uses: actions/setup-node@v4\n        with:\n          node-version: '%s'\n          cache: 'npm'\n", ver))
		b.WriteString("      - run: npm ci\n")
		b.WriteString("      - run: npm run lint --if-present\n")
		b.WriteString("      - run: npm test\n")
		b.WriteString("      - run: npm run build --if-present\n")

	case LangPython:
		ver := defaultIfEmpty(req.PythonVersion, "3.12")
		b.WriteString(fmt.Sprintf("      - uses: actions/setup-python@v5\n        with:\n          python-version: '%s'\n", ver))
		b.WriteString("      - run: pip install -r requirements.txt\n")
		b.WriteString("      - run: pip install ruff pytest\n")
		b.WriteString("      - run: ruff check .\n")
		b.WriteString("      - run: pytest --tb=short\n")

	case LangRust:
		b.WriteString("      - uses: dtolnay/rust-toolchain@stable\n")
		b.WriteString("        with:\n          components: clippy, rustfmt\n")
		b.WriteString("      - run: cargo fmt --check\n")
		b.WriteString("      - run: cargo clippy -- -D warnings\n")
		b.WriteString("      - run: cargo test\n")
		b.WriteString("      - run: cargo build --release\n")
	}
}

func writeGHASecuritySteps(b *strings.Builder) {
	b.WriteString("\n  security:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    needs: build\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")
	b.WriteString("      - name: Run gitleaks secret scanning\n")
	b.WriteString("        uses: gitleaks/gitleaks-action@v2\n")
	b.WriteString("        env:\n")
	b.WriteString("          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
	b.WriteString("      - name: Run semgrep SAST\n")
	b.WriteString("        uses: returntocorp/semgrep-action@v1\n")
	b.WriteString("        with:\n")
	b.WriteString("          config: auto\n")
}

func writeGHADockerSteps(b *strings.Builder, req PipelineRequest) {
	registry := defaultIfEmpty(req.DockerRegistry, "ghcr.io")

	b.WriteString("\n  docker:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    needs: build\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")
	b.WriteString("      - uses: docker/setup-buildx-action@v3\n")
	b.WriteString("      - uses: docker/login-action@v3\n")
	b.WriteString("        with:\n")
	b.WriteString(fmt.Sprintf("          registry: %s\n", registry))
	b.WriteString("          username: ${{ github.actor }}\n")
	b.WriteString("          password: ${{ secrets.GITHUB_TOKEN }}\n")
	b.WriteString("      - uses: docker/build-push-action@v5\n")
	b.WriteString("        with:\n")
	b.WriteString("          push: true\n")
	b.WriteString(fmt.Sprintf("          tags: %s/%s:${{ github.sha }}\n", registry, req.ProjectName))
}

// ---------------------------------------------------------------------------
// GitLab CI
// ---------------------------------------------------------------------------

func generateGitLabCI(req PipelineRequest) (PipelineResult, error) {
	var b strings.Builder

	b.WriteString("stages:\n  - lint\n  - test\n  - build\n")

	if req.IncludeSecurityGates {
		b.WriteString("  - security\n")
	}
	if req.IncludeDocker {
		b.WriteString("  - docker\n")
	}
	b.WriteString("\n")

	writeGitLabLanguageJobs(&b, req)

	if req.IncludeSecurityGates {
		writeGitLabSecurityJobs(&b)
	}

	if req.IncludeDocker {
		writeGitLabDockerJob(&b, req)
	}

	return PipelineResult{
		Platform: PlatformGitLabCI,
		Filename: ".gitlab-ci.yml",
		Content:  b.String(),
	}, nil
}

func writeGitLabLanguageJobs(b *strings.Builder, req PipelineRequest) {
	switch req.Language {
	case LangGo:
		ver := defaultIfEmpty(req.GoVersion, "1.22")
		b.WriteString(fmt.Sprintf("default:\n  image: golang:%s\n\n", ver))
		b.WriteString("lint:\n  stage: lint\n  script:\n    - go vet ./...\n\n")
		b.WriteString("test:\n  stage: test\n  script:\n    - go test -race -coverprofile=coverage.out ./...\n")
		b.WriteString("  artifacts:\n    paths:\n      - coverage.out\n\n")
		b.WriteString("build:\n  stage: build\n  script:\n    - go build -o bin/ ./...\n")
		b.WriteString("  artifacts:\n    paths:\n      - bin/\n\n")

	case LangNode:
		ver := defaultIfEmpty(req.NodeVersion, "20")
		b.WriteString(fmt.Sprintf("default:\n  image: node:%s\n\n", ver))
		b.WriteString("lint:\n  stage: lint\n  script:\n    - npm ci\n    - npm run lint --if-present\n\n")
		b.WriteString("test:\n  stage: test\n  script:\n    - npm ci\n    - npm test\n\n")
		b.WriteString("build:\n  stage: build\n  script:\n    - npm ci\n    - npm run build --if-present\n\n")

	case LangPython:
		ver := defaultIfEmpty(req.PythonVersion, "3.12")
		b.WriteString(fmt.Sprintf("default:\n  image: python:%s\n\n", ver))
		b.WriteString("lint:\n  stage: lint\n  script:\n    - pip install ruff\n    - ruff check .\n\n")
		b.WriteString("test:\n  stage: test\n  script:\n    - pip install -r requirements.txt\n    - pip install pytest\n    - pytest --tb=short\n\n")
		b.WriteString("build:\n  stage: build\n  script:\n    - echo 'Build step for Python project'\n\n")

	case LangRust:
		b.WriteString("default:\n  image: rust:latest\n\n")
		b.WriteString("lint:\n  stage: lint\n  script:\n    - rustup component add clippy rustfmt\n    - cargo fmt --check\n    - cargo clippy -- -D warnings\n\n")
		b.WriteString("test:\n  stage: test\n  script:\n    - cargo test\n\n")
		b.WriteString("build:\n  stage: build\n  script:\n    - cargo build --release\n")
		b.WriteString("  artifacts:\n    paths:\n      - target/release/\n\n")
	}
}

func writeGitLabSecurityJobs(b *strings.Builder) {
	b.WriteString("gitleaks:\n  stage: security\n  image: zricethezav/gitleaks:latest\n")
	b.WriteString("  script:\n    - gitleaks detect --source . --verbose\n\n")
	b.WriteString("semgrep:\n  stage: security\n  image: returntocorp/semgrep:latest\n")
	b.WriteString("  script:\n    - semgrep --config auto .\n\n")
}

func writeGitLabDockerJob(b *strings.Builder, req PipelineRequest) {
	registry := defaultIfEmpty(req.DockerRegistry, "registry.gitlab.com")

	b.WriteString("docker-build:\n  stage: docker\n")
	b.WriteString("  image: docker:latest\n")
	b.WriteString("  services:\n    - docker:dind\n")
	b.WriteString("  script:\n")
	b.WriteString(fmt.Sprintf("    - docker build -t %s/%s:$CI_COMMIT_SHA .\n", registry, req.ProjectName))
	b.WriteString(fmt.Sprintf("    - docker push %s/%s:$CI_COMMIT_SHA\n", registry, req.ProjectName))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func defaultIfEmpty(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}
