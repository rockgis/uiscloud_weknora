import { get, post, put, del } from '../../utils/request';

export interface ModelConfig {
  id?: string;
  tenant_id?: number;
  name: string;
  type: 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM';
  source: 'local' | 'remote';
  description?: string;
  parameters: {
    base_url?: string;
    api_key?: string;
    embedding_parameters?: {
      dimension?: number;
      truncate_prompt_tokens?: number;
    };
    interface_type?: 'ollama' | 'openai';
    parameter_size?: string;
  };
  is_default?: boolean;
  is_builtin?: boolean;
  status?: string;
  created_at?: string;
  updated_at?: string;
  deleted_at?: string | null;
}

export function createModel(data: ModelConfig): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    post('/api/v1/models', data)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || '모델 생성 실패'));
        }
      })
      .catch((error: any) => {
        console.error('모델 생성 실패:', error);
        reject(error);
      });
  });
}

export function listModels(type?: string): Promise<ModelConfig[]> {
  return new Promise((resolve, reject) => {
    const url = `/api/v1/models`;
    get(url)
      .then((response: any) => {
        if (response.success && response.data) {
          if (type) {
            response.data = response.data.filter((item: ModelConfig) => item.type === type);
          }
          resolve(response.data);
        } else {
          resolve([]);
        }
      })
      .catch((error: any) => {
        console.error('모델 목록 조회 실패:', error);
        resolve([]);
      });
  });
}

export function getModel(id: string): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    get(`/api/v1/models/${id}`)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || '모델 조회 실패'));
        }
      })
      .catch((error: any) => {
        console.error('모델 조회 실패:', error);
        reject(error);
      });
  });
}

export function updateModel(id: string, data: Partial<ModelConfig>): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    put(`/api/v1/models/${id}`, data)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || '모델 업데이트 실패'));
        }
      })
      .catch((error: any) => {
        console.error('모델 업데이트 실패:', error);
        reject(error);
      });
  });
}

export function deleteModel(id: string): Promise<void> {
  return new Promise((resolve, reject) => {
    del(`/api/v1/models/${id}`)
      .then((response: any) => {
        if (response.success) {
          resolve();
        } else {
          reject(new Error(response.message || '모델 삭제 실패'));
        }
      })
      .catch((error: any) => {
        console.error('모델 삭제 실패:', error);
        reject(error);
      });
  });
}

