## Plan: Comprehensive Auth Service Enhancement

Implement OAuth authentication, API documentation, comprehensive testing, structural refactoring, Docker containerization, multi-environment configuration, and deployment setup for the Go auth service.

**Steps**

### Phase 1: OAuth Integration (Facebook & Google)
1. Add OAuth dependencies: golang.org/x/oauth2, google.golang.org/api/oauth2/v2, github.com/markbates/goth
2. Create OAuth service in internal/service/oauth_service.go with provider configs and user linking logic
3. Add OAuth handlers in internal/handler/oauth.go for /auth/{provider} and /auth/{provider}/callback endpoints
4. Update user model to include OAuth provider fields (provider, provider_id)
5. Create migration for OAuth fields in users table
6. Register OAuth routes in router/router.go
7. Update environment config for OAuth client IDs and secrets

### Phase 2: API Documentation (Swagger)
1. Install Swaggo CLI and add gin-swagger dependency
2. Add Swagger annotations to all handler functions in internal/handler/auth.go and qr.go
3. Run swag init to generate docs/swagger.json and docs/swagger.yaml
4. Register Swagger UI routes (/swagger/*any) in router/router.go
5. Test Swagger UI at http://localhost:8080/swagger/index.html

### Phase 3: Unit Test Coverage
1. Create service layer tests: user_service_test.go, token_service_test.go, qr_service_test.go, email_service_test.go
2. Create repository layer tests: user_repository_test.go, auth_code_repository_test.go with test database setup
3. Add middleware tests: auth_middleware_test.go
4. Complete handler tests for missing endpoints: ConfirmEmail and ResendConfirmationEmail
5. Add test utilities and fixtures for common test data
6. Run go test -cover to verify >70% coverage

### Phase 4: Folder Structure Refactoring
1. Create new structure: internal/config/, internal/domain/, internal/errors/, internal/repository/interfaces.go, internal/transport/http/, internal/transport/ws/
2. Move and rename files: model/ to domain/, websocket/ to transport/ws/, router/ to transport/http/
3. Add repository interfaces and update services to use them
4. Implement centralized config in internal/config/config.go
5. Add custom error types in internal/errors/errors.go
6. Update main.go to use new structure and dependency injection
7. Move JWT logic to pkg/jwt/manager.go with interface

### Phase 5: Docker Containerization
1. Create Dockerfile with multi-stage build
2. Create docker-compose.yml with postgres and auth-service services
3. Add .env.docker with container-specific environment variables
4. Update config to support DATABASE_URL for container networking
5. Test docker-compose up and verify service runs in containers

### Phase 6: Environment Configurations
1. Enhance config loading for APP_ENV (development, production, testing)
2. Create separate .env files: .env.development, .env.production, .env.testing
3. Add test database configuration for unit tests
4. Implement environment-specific logging and error handling
5. Add config validation and defaults

### Phase 7: Deployment
1. Set up CI/CD pipeline (GitHub Actions) for automated testing and building
2. Create deployment manifests for chosen platform (e.g., Kubernetes, Docker Compose for cloud)
3. Configure production environment variables and secrets management
4. Add health checks and monitoring endpoints
5. Document deployment process in README.md

**Relevant files**
- internal/service/oauth_service.go — New OAuth service
- internal/handler/oauth.go — New OAuth handlers
- internal/handler/auth.go — Add Swagger annotations
- docs/ — Generated Swagger docs
- internal/service/*_test.go — New service tests
- internal/repository/*_test.go — New repository tests
- internal/config/config.go — Centralized config
- internal/domain/ — Refactored models
- Dockerfile — Container build
- docker-compose.yml — Local orchestration
- .env.* — Environment configs
- .github/workflows/ — CI/CD pipeline

**Verification**
1. OAuth: Test login flows with Facebook/Google accounts, verify user creation/linking
2. API Docs: Access Swagger UI, verify all endpoints documented with try-it-out functionality
3. Tests: Run go test -cover, achieve >70% coverage, all tests pass
4. Refactoring: Build and run service, verify no functionality broken
5. Docker: docker-compose up succeeds, service accessible at localhost:8080
6. Environments: Switch APP_ENV, verify different configs loaded
7. Deployment: Deploy to staging, verify service runs in production-like environment

**Decisions**
- OAuth providers: Facebook and Google using goth library for unified interface
- API docs: Swaggo for annotation-based OpenAPI generation
- Testing: Add service and repository unit tests to reach good coverage
- Structure: Adopt standard Go layout with domain, transport layers
- Docker: Multi-stage build with Alpine, Compose for local dev
- Environments: Separate .env files with APP_ENV variable
- Deployment: Assume cloud-native with Kubernetes, but provide general setup

**Further Considerations**
1. OAuth: How to handle user account linking if email already exists?
2. Deployment: Which cloud platform (AWS, GCP, Azure) or on-prem?
3. Testing: Integration tests for full request flows?
4. Security: Rate limiting, CORS, input validation enhancements?
