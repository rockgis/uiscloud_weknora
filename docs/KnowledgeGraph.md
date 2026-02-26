# WeKnora 지식 그래프

## 빠른 시작

- .env에서 관련 환경 변수 설정
    - Neo4j 활성화: `NEO4J_ENABLE=true`
    - Neo4j URI: `NEO4J_URI=bolt://neo4j:7687`
    - Neo4j 사용자 이름: `NEO4J_USERNAME=neo4j`
    - Neo4j 비밀번호: `NEO4J_PASSWORD=password`

- Neo4j 시작
```bash
docker-compose --profile neo4j up -d
```

- 지식베이스 설정 페이지에서 엔티티 및 관계 추출을 활성화하고 안내에 따라 관련 내용을 설정하세요.

## 그래프 생성

임의의 문서를 업로드하면 시스템이 자동으로 엔티티와 관계를 추출하여 해당 지식 그래프를 생성합니다.

![지식 그래프 예시](./images/graph3.png)

## 그래프 확인

`http://localhost:7474`에 로그인하여 `match (n) return (n)`을 실행하면 생성된 지식 그래프를 확인할 수 있습니다.

대화 중 시스템이 자동으로 지식 그래프를 조회하여 관련 지식을 가져옵니다.
