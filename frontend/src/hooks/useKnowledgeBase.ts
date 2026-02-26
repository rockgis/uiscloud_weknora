import { ref, reactive } from "vue";
import { storeToRefs } from "pinia";
import { formatStringDate, kbFileTypeVerification } from "../utils/index";
import { MessagePlugin } from "tdesign-vue-next";
import {
  uploadKnowledgeFile,
  listKnowledgeFiles,
  getKnowledgeDetails,
  delKnowledgeDetails,
  getKnowledgeDetailsCon,
} from "@/api/knowledge-base/index";
import { knowledgeStore } from "@/stores/knowledge";
import { useRoute } from 'vue-router';

const usemenuStore = knowledgeStore();
export default function (knowledgeBaseId?: string) {
  const route = useRoute();
  const { cardList, total } = storeToRefs(usemenuStore);
  let moreIndex = ref(-1);
  const details = reactive({
    title: "",
    time: "",
    md: [] as any[],
    id: "",
    total: 0,
    type: "",
    source: "",
    file_type: ""
  });
  const getKnowled = (
    query: { page: number; page_size: number; tag_id?: string; keyword?: string; file_type?: string } = { page: 1, page_size: 35 },
    kbId?: string,
  ) => {
    const targetKbId = kbId || knowledgeBaseId;
    if (!targetKbId) return;
    
    listKnowledgeFiles(targetKbId, query)
      .then((result: any) => {
        const { data, total: totalResult } = result;
    const cardList_ = data.map((item: any) => {
      const rawName = item.file_name || item.title || item.source || '이름 없는 문서'
      const dotIndex = rawName.lastIndexOf('.')
      const displayName = dotIndex > 0 ? rawName.substring(0, dotIndex) : rawName
      const fileTypeSource = item.file_type || (item.type === 'manual' ? 'MANUAL' : '')
      return {
        ...item,
        original_file_name: item.file_name,
        display_name: displayName,
        file_name: displayName,
        updated_at: formatStringDate(new Date(item.updated_at)),
        isMore: false,
        file_type: fileTypeSource ? String(fileTypeSource).toLocaleUpperCase() : '',
      }
    });
        
        if (query.page === 1) {
          cardList.value = cardList_;
        } else {
          cardList.value.push(...cardList_);
        }
        total.value = totalResult;
      })
      .catch(() => {});
  };
  const delKnowledge = (index: number, item: any) => {
    cardList.value[index].isMore = false;
    moreIndex.value = -1;
    delKnowledgeDetails(item.id)
      .then((result: any) => {
        if (result.success) {
          MessagePlugin.info("지식이 삭제되었습니다!");
          getKnowled();
        } else {
          MessagePlugin.error("지식 삭제에 실패했습니다!");
        }
      })
      .catch(() => {
        MessagePlugin.error("지식 삭제에 실패했습니다!");
      });
  };
  const openMore = (index: number) => {
    moreIndex.value = index;
  };
  const onVisibleChange = (visible: boolean) => {
    if (!visible) {
      moreIndex.value = -1;
    }
  };
  const requestMethod = (file: any, uploadInput: any) => {
    if (!(file instanceof File) || !uploadInput) {
      MessagePlugin.error("지원하지 않는 파일 형식입니다!");
      return;
    }
    
    if (kbFileTypeVerification(file)) {
      return;
    }
    
    let currentKbId: string | undefined = (route.params as any)?.kbId as string;
    if (!currentKbId && typeof window !== 'undefined') {
      const match = window.location.pathname.match(/knowledge-bases\/([^/]+)/);
      if (match?.[1]) currentKbId = match[1];
    }
    if (!currentKbId) {
      currentKbId = knowledgeBaseId;
    }
    if (!currentKbId) {
      MessagePlugin.error("지식베이스 ID가 없습니다");
      return;
    }
    
    uploadKnowledgeFile(currentKbId, { file })
      .then((result: any) => {
        if (result.success) {
          MessagePlugin.info("업로드 성공!");
          getKnowled({ page: 1, page_size: 35 }, currentKbId);
        } else {
          const errorMessage = result.error?.message || result.message || "업로드 실패!";
          MessagePlugin.error(result.code === 'duplicate_file' ? "파일이 이미 존재합니다" : errorMessage);
        }
        uploadInput.value.value = "";
      })
      .catch((err: any) => {
        const errorMessage = err.error?.message || err.message || "업로드 실패!";
        MessagePlugin.error(err.code === 'duplicate_file' ? "파일이 이미 존재합니다" : errorMessage);
        uploadInput.value.value = "";
      });
  };
  const getCardDetails = (item: any) => {
    Object.assign(details, {
      title: "",
      time: "",
      md: [],
      id: "",
      type: "",
      source: "",
      file_type: ""
    });
    getKnowledgeDetails(item.id)
      .then((result: any) => {
        if (result.success && result.data) {
          const { data } = result;
          Object.assign(details, {
            title: data.file_name || data.title || data.source || '이름 없는 문서',
            time: formatStringDate(new Date(data.updated_at)),
            id: data.id,
            type: data.type || 'file',
            source: data.source || '',
            file_type: data.file_type || ''
          });
        }
      })
      .catch(() => {});
    getfDetails(item.id, 1);
  };
  
  const getfDetails = (id: string, page: number) => {
    getKnowledgeDetailsCon(id, page)
      .then((result: any) => {
        if (result.success && result.data) {
          const { data, total: totalResult } = result;
          if (page === 1) {
            details.md = data;
          } else {
            details.md.push(...data);
          }
          details.total = totalResult;
        }
      })
      .catch(() => {});
  };
  return {
    cardList,
    moreIndex,
    getKnowled,
    details,
    delKnowledge,
    openMore,
    onVisibleChange,
    requestMethod,
    getCardDetails,
    total,
    getfDetails,
  };
}
