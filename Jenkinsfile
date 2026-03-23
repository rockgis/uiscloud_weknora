// WeKnora Jenkins CI/CD 파이프라인
// 사전 요구사항: Jenkins Credentials 설정 필요 (jenkins/README.md 참고)

pipeline {
    agent any

    // ──────────────────────────────────────────────
    // 빌드 파라미터
    // ──────────────────────────────────────────────
    parameters {
        choice(
            name: 'DEPLOY_ENV',
            choices: ['dev', 'staging', 'prod'],
            description: '배포 환경 선택'
        )
        booleanParam(
            name: 'BUILD_DOCREADER',
            defaultValue: false,
            description: 'Docreader 이미지 빌드 여부 (Python 의존성으로 빌드 시간 길음)'
        )
        booleanParam(
            name: 'SKIP_TESTS',
            defaultValue: false,
            description: '테스트 단계 건너뛰기 (긴급 배포 시에만 사용)'
        )
        booleanParam(
            name: 'DEPLOY_ONLY',
            defaultValue: false,
            description: '빌드 없이 현재 이미지로 배포만 수행'
        )
    }

    // ──────────────────────────────────────────────
    // 환경 변수
    // Jenkins > Manage Jenkins > Credentials 에서 설정 필요
    // ──────────────────────────────────────────────
    environment {
        // Docker 레지스트리 (예: registry.example.com 또는 docker.io)
        DOCKER_REGISTRY  = credentials('weknora-docker-registry-url')
        DOCKER_CREDS     = credentials('weknora-docker-credentials')

        // 이미지 이름
        IMAGE_APP        = "${DOCKER_REGISTRY}/weknora-app"
        IMAGE_DOCREADER  = "${DOCKER_REGISTRY}/weknora-docreader"
        IMAGE_UI         = "${DOCKER_REGISTRY}/weknora-ui"

        // 버전 정보 (빌드 시 자동 추출)
        APP_VERSION      = ''
        COMMIT_ID        = ''
        BUILD_TIME       = ''
        GO_VERSION       = ''
    }

    // ──────────────────────────────────────────────
    // 빌드 옵션
    // ──────────────────────────────────────────────
    options {
        buildDiscarder(logRotator(numToKeepStr: '20'))
        timeout(time: 90, unit: 'MINUTES')
        disableConcurrentBuilds()
        timestamps()
    }

    stages {

        // ──────────────────────────────────────────
        // 1. 소스 체크아웃
        // ──────────────────────────────────────────
        stage('Checkout') {
            steps {
                checkout scm
                script {
                    env.APP_VERSION = sh(script: 'cat VERSION', returnStdout: true).trim()
                    env.COMMIT_ID   = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                    env.BUILD_TIME  = sh(script: 'date -u +"%Y-%m-%dT%H:%M:%SZ"', returnStdout: true).trim()
                    env.GO_VERSION  = sh(script: 'go version 2>/dev/null | awk \'{print $3}\' || echo "unknown"', returnStdout: true).trim()

                    echo "======================================"
                    echo "버전     : ${env.APP_VERSION}"
                    echo "커밋     : ${env.COMMIT_ID}"
                    echo "빌드시간 : ${env.BUILD_TIME}"
                    echo "Go 버전  : ${env.GO_VERSION}"
                    echo "배포환경 : ${params.DEPLOY_ENV}"
                    echo "======================================"
                }
            }
        }

        // ──────────────────────────────────────────
        // 2. 코드 품질 검사
        // ──────────────────────────────────────────
        stage('Lint') {
            when {
                not { expression { params.DEPLOY_ONLY } }
            }
            parallel {
                stage('Go Lint') {
                    steps {
                        sh '''
                            if command -v golangci-lint &> /dev/null; then
                                golangci-lint run --timeout=5m --new-from-rev=HEAD~1
                            else
                                echo "golangci-lint 미설치 — 스킵"
                            fi
                        '''
                    }
                }
                stage('Frontend Type Check') {
                    steps {
                        dir('frontend') {
                            sh '''
                                if command -v pnpm &> /dev/null; then
                                    pnpm install --frozen-lockfile
                                    pnpm type-check
                                elif command -v npm &> /dev/null; then
                                    npm ci
                                    npm run type-check
                                else
                                    echo "pnpm/npm 미설치 — 스킵"
                                fi
                            '''
                        }
                    }
                }
            }
        }

        // ──────────────────────────────────────────
        // 3. 테스트
        // ──────────────────────────────────────────
        stage('Test') {
            when {
                allOf {
                    not { expression { params.SKIP_TESTS } }
                    not { expression { params.DEPLOY_ONLY } }
                }
            }
            steps {
                sh '''
                    go test -v -race -coverprofile=coverage.out \
                        $(go list ./... | grep -v "docreader/client" | grep -v "e2e") \
                        2>&1 | tee test-output.txt
                '''
                sh 'go tool cover -func=coverage.out | tail -1'
            }
            post {
                always {
                    // 테스트 결과 아카이브
                    archiveArtifacts artifacts: 'coverage.out', allowEmptyArchive: true
                }
            }
        }

        // ──────────────────────────────────────────
        // 4. Docker 이미지 빌드 (병렬)
        // ──────────────────────────────────────────
        stage('Build Images') {
            when {
                not { expression { params.DEPLOY_ONLY } }
            }
            parallel {

                stage('Build App') {
                    steps {
                        sh """
                            docker build \
                                -f docker/Dockerfile.app \
                                --build-arg VERSION_ARG=${env.APP_VERSION} \
                                --build-arg COMMIT_ID_ARG=${env.COMMIT_ID} \
                                --build-arg BUILD_TIME_ARG=${env.BUILD_TIME} \
                                --build-arg GO_VERSION_ARG=${env.GO_VERSION} \
                                -t ${env.IMAGE_APP}:${env.APP_VERSION} \
                                -t ${env.IMAGE_APP}:latest \
                                .
                        """
                    }
                }

                stage('Build Frontend') {
                    steps {
                        sh """
                            docker build \
                                -f frontend/Dockerfile \
                                -t ${env.IMAGE_UI}:${env.APP_VERSION} \
                                -t ${env.IMAGE_UI}:latest \
                                frontend/
                        """
                    }
                }

                stage('Build Docreader') {
                    when {
                        expression { params.BUILD_DOCREADER }
                    }
                    steps {
                        sh """
                            TARGETARCH=\$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
                            docker build \
                                -f docker/Dockerfile.docreader \
                                --build-arg TARGETARCH=\${TARGETARCH} \
                                -t ${env.IMAGE_DOCREADER}:${env.APP_VERSION} \
                                -t ${env.IMAGE_DOCREADER}:latest \
                                .
                        """
                    }
                }
            }
        }

        // ──────────────────────────────────────────
        // 5. 레지스트리 푸시
        // ──────────────────────────────────────────
        stage('Push Images') {
            when {
                not { expression { params.DEPLOY_ONLY } }
            }
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'weknora-docker-credentials',
                    usernameVariable: 'DOCKER_USER',
                    passwordVariable: 'DOCKER_PASS'
                )]) {
                    sh 'echo "$DOCKER_PASS" | docker login ${DOCKER_REGISTRY} -u "$DOCKER_USER" --password-stdin'
                }

                sh "docker push ${env.IMAGE_APP}:${env.APP_VERSION}"
                sh "docker push ${env.IMAGE_APP}:latest"
                sh "docker push ${env.IMAGE_UI}:${env.APP_VERSION}"
                sh "docker push ${env.IMAGE_UI}:latest"

                script {
                    if (params.BUILD_DOCREADER) {
                        sh "docker push ${env.IMAGE_DOCREADER}:${env.APP_VERSION}"
                        sh "docker push ${env.IMAGE_DOCREADER}:latest"
                    }
                }
            }
            post {
                always {
                    sh 'docker logout ${DOCKER_REGISTRY} || true'
                }
            }
        }

        // ──────────────────────────────────────────
        // 6. 배포
        // 환경별 SSH Credentials: weknora-ssh-{dev|staging|prod}
        // ──────────────────────────────────────────
        stage('Deploy') {
            steps {
                script {
                    def sshCredId   = "weknora-ssh-${params.DEPLOY_ENV}"
                    def deployHost  = getDeployHost(params.DEPLOY_ENV)
                    def deployPath  = getDeployPath(params.DEPLOY_ENV)
                    def imageTag    = params.DEPLOY_ONLY ? 'latest' : env.APP_VERSION

                    echo "배포 대상: ${deployHost} (${params.DEPLOY_ENV})"
                    echo "배포 경로: ${deployPath}"
                    echo "이미지 태그: ${imageTag}"

                    // deploy.sh 업로드 후 실행
                    sshagent(credentials: [sshCredId]) {
                        sh """
                            scp -o StrictHostKeyChecking=no \
                                jenkins/deploy.sh \
                                deploy@${deployHost}:${deployPath}/deploy.sh
                        """
                        sh """
                            ssh -o StrictHostKeyChecking=no deploy@${deployHost} \
                                "chmod +x ${deployPath}/deploy.sh && \
                                 IMAGE_TAG=${imageTag} \
                                 DEPLOY_ENV=${params.DEPLOY_ENV} \
                                 DOCKER_REGISTRY=${env.DOCKER_REGISTRY} \
                                 ${deployPath}/deploy.sh"
                        """
                    }
                }
            }
        }

    } // end stages

    // ──────────────────────────────────────────────
    // 빌드 후 처리
    // ──────────────────────────────────────────────
    post {
        always {
            // 로컬 Docker 이미지 정리 (디스크 절약)
            sh '''
                docker rmi $(docker images -f "dangling=true" -q) 2>/dev/null || true
            '''
        }
        success {
            echo "배포 성공: ${params.DEPLOY_ENV} 환경 v${env.APP_VERSION} (${env.COMMIT_ID})"
        }
        failure {
            echo "배포 실패: 로그를 확인하세요."
        }
    }

} // end pipeline


// ──────────────────────────────────────────────────
// 헬퍼 함수: 환경별 배포 서버 호스트
// Jenkins > Manage Jenkins > Configure System 에서
// 환경변수로 관리하거나 아래 값을 직접 수정하세요.
// ──────────────────────────────────────────────────
def getDeployHost(String env) {
    def hosts = [
        dev     : System.getenv('WEKNORA_DEV_HOST')     ?: 'dev.example.com',
        staging : System.getenv('WEKNORA_STAGING_HOST') ?: 'staging.example.com',
        prod    : System.getenv('WEKNORA_PROD_HOST')    ?: 'prod.example.com',
    ]
    return hosts[env] ?: error("알 수 없는 배포 환경: ${env}")
}

def getDeployPath(String env) {
    def paths = [
        dev     : '/opt/weknora-dev',
        staging : '/opt/weknora-staging',
        prod    : '/opt/weknora',
    ]
    return paths[env] ?: '/opt/weknora'
}
