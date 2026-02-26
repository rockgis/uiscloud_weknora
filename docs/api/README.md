# WeKnora API 문서

## 목차

- [개요](#개요)
- [기본 정보](#기본-정보)
- [인증 방식](#인증-방식)
- [오류 처리](#오류-처리)
- [API 개요](#api-개요)

## 개요

WeKnora는 지식베이스 생성 및 관리, 지식 검색, 그리고 지식 기반 질의응답을 위한 일련의 RESTful API를 제공합니다. 본 문서에서는 이러한 API의 사용 방법을 상세히 설명합니다.

## 기본 정보

- **기본 URL**: `/api/v1`
- **응답 형식**: JSON
- **인증 방식**: API Key

## 인증 방식

모든 API 요청은 HTTP 요청 헤더에 `X-API-Key`를 포함하여 인증해야 합니다:

```
X-API-Key: your_api_key
```

문제 추적 및 디버깅을 용이하게 하기 위해, 각 요청의 HTTP 요청 헤더에 `X-Request-ID`를 추가하는 것을 권장합니다:

```
X-Request-ID: unique_request_id
```

### API Key 발급

웹 페이지에서 계정 등록을 완료한 후, 계정 정보 페이지에서 API Key를 발급받으세요.

API Key는 안전하게 보관하시고 외부에 노출되지 않도록 주의하세요. API Key는 귀하의 계정을 식별하며, 전체 API 접근 권한을 가집니다.

## 오류 처리

모든 API는 표준 HTTP 상태 코드를 사용하여 요청 상태를 나타내며, 통일된 오류 응답 형식을 반환합니다:

```json
{
  "success": false,
  "error": {
    "code": "오류 코드",
    "message": "오류 메시지",
    "details": "오류 상세 내용"
  }
}
```

## API 개요

WeKnora API는 기능별로 다음과 같이 분류됩니다:

| 분류 | 설명 | 문서 링크 |
|------|------|----------|
| 테넌트 관리 | 테넌트 계정 생성 및 관리 | [tenant.md](./tenant.md) |
| 지식베이스 관리 | 지식베이스 생성, 조회 및 관리 | [knowledge-base.md](./knowledge-base.md) |
| 지식 관리 | 지식 콘텐츠 업로드, 검색 및 관리 | [knowledge.md](./knowledge.md) |
| 모델 관리 | 다양한 AI 모델 설정 및 관리 | [model.md](./model.md) |
| 청크 관리 | 지식의 청크 콘텐츠 관리 | [chunk.md](./chunk.md) |
| 태그 관리 | 지식베이스의 태그 분류 관리 | [tag.md](./tag.md) |
| FAQ 관리 | FAQ 질의응답 쌍 관리 | [faq.md](./faq.md) |
| 세션 관리 | 대화 세션 생성 및 관리 | [session.md](./session.md) |
| 채팅 기능 | 지식베이스 및 Agent 기반 질의응답 | [chat.md](./chat.md) |
| 메시지 관리 | 대화 메시지 조회 및 관리 | [message.md](./message.md) |
| 평가 기능 | 모델 성능 평가 | [evaluation.md](./evaluation.md) |
