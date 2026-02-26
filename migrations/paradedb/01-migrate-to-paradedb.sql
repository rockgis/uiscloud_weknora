
-- pg_dump -U postgres -h localhost -p 5432 -d your_database > backup.sql

-- psql -U postgres -h localhost -p 5432 -d your_database < backup.sql



-- Insert some sample data
-- INSERT INTO tenants (id, name, description, status, api_key)
-- VALUES 
--     (1, 'Demo Tenant', 'This is a demo tenant for testing', 'active', 'sk-00000001abcdefg123456')
-- ON CONFLICT DO NOTHING;

-- SELECT setval('tenants_id_seq', (SELECT MAX(id) FROM tenants));


-- -- Create knowledge base
-- INSERT INTO knowledge_bases (id, name, description, tenant_id, chunking_config, image_processing_config, embedding_model_id)
-- VALUES 
--     ('kb-00000001', 'Default Knowledge Base', 'Default knowledge base for testing', 1, '{"chunk_size": 512, "chunk_overlap": 50, "separators": ["\n\n", "\n", "。"], "keep_separator": true}', '{"enable_multimodal": false, "model_id": ""}', 'model-embedding-00000001'),
--     ('kb-00000002', 'Test Knowledge Base', 'Test knowledge base for development', 1, '{"chunk_size": 512, "chunk_overlap": 50, "separators": ["\n\n", "\n", "。"], "keep_separator": true}', '{"enable_multimodal": false, "model_id": ""}', 'model-embedding-00000001'),
--     ('kb-00000003', 'Test Knowledge Base 2', 'Test knowledge base for development 2', 1, '{"chunk_size": 512, "chunk_overlap": 50, "separators": ["\n\n", "\n", "。"], "keep_separator": true}', '{"enable_multimodal": false, "model_id": ""}', 'model-embedding-00000001')
-- ON CONFLICT DO NOTHING;


SELECT COUNT(*) FROM tenants;
SELECT COUNT(*) FROM models;
SELECT COUNT(*) FROM knowledge_bases;
SELECT COUNT(*) FROM knowledges;



CREATE TABLE chinese_documents (
    id SERIAL PRIMARY KEY,
    title TEXT,
    content TEXT,
    published_date DATE
);

CREATE INDEX idx_documents_bm25 ON chinese_documents
USING bm25 (id, content)
WITH (
    key_field = 'id',
    text_fields = '{
        "content": {
          "tokenizer": {"type": "chinese_lindera"}
        }
    }'
);

INSERT INTO chinese_documents (title, content, published_date)
VALUES
('인공지능의 발전', 'AI 기술이 빠르게 발전하고 있으며 우리 생활 전반에 영향을 미치고 있습니다. 대형 언어 모델은 최근의 중요한 돌파구입니다.', '2023-01-15'),
('머신러닝 기초', '머신러닝은 인공지능의 중요한 분야로, 지도학습, 비지도학습, 강화학습 등의 방법을 포함합니다.', '2023-02-20'),
('딥러닝 응용', '딥러닝은 이미지 인식, 자연어 처리, 음성 인식 등 다양한 분야에 광범위하게 활용됩니다.', '2023-03-10'),
('자연어 처리 기술', '자연어 처리는 컴퓨터가 인간의 언어를 이해하고 해석하며 생성할 수 있게 하는 AI의 핵심 기술입니다.', '2023-04-05'),
('컴퓨터 비전 입문', '컴퓨터 비전은 기계가 시각 세계를 인식하고 이해할 수 있게 하며, 보안, 의료 등 분야에 널리 활용됩니다.', '2023-05-12');

INSERT INTO chinese_documents (title, content, published_date)
VALUES 
('hello world', 'hello world', '2023-05-12');
