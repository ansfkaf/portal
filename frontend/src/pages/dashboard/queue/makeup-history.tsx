// src/pages/dashboard/makeup-history.tsx
import { useEffect, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import { DataTable, type Column } from 'src/components/table/data-table';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import axiosInstance from 'src/lib/axios';

// 补机历史记录接口定义
interface MakeupHistoryItem {
  user_id: string;
  region: string; // 新增区域字段
  count: number;
  timestamp: string;
}

// API响应接口
interface MakeupHistoryResponse {
  code: number;
  message: string;
  data: {
    list: MakeupHistoryItem[];
    total: number;
  };
}

export default function MakeupHistoryPage() {
  // 状态管理
  const [historyData, setHistoryData] = useState<MakeupHistoryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const { addAlert } = useAlert();

  // 获取补机历史数据
  const fetchMakeupHistory = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.post<MakeupHistoryResponse>('/monitor/makeup-history');
      if (response.data.code === 200) {
        // 确保historyData始终是数组，即使API返回null或undefined
        setHistoryData(response.data.data.list || []);
      } else {
        addAlert('error', '获取补机历史数据失败');
        // 设置为空数组而不是null
        setHistoryData([]);
      }
    } catch (error) {
      addAlert('error', '获取补机历史数据请求失败');
      // 出错时也确保设置为空数组
      setHistoryData([]);
    } finally {
      setLoading(false);
    }
  };

  // 初始加载数据
  useEffect(() => {
    fetchMakeupHistory();
  }, []);
  
  // 获取区域的显示名称
  const getRegionName = (region: string): string => {
    switch (region) {
      case 'ap-east-1':
        return '香港区';
      case 'ap-northeast-3':
        return '日本区';
      case 'ap-southeast-1':
        return '新加坡区';
      default:
        return region || '香港区';
    }
  };

  // 表格列定义
  const columns: Column[] = [
    { 
      id: 'user_id', 
      label: '用户ID', 
      sortable: true,
      width: '100px'
    },
    { 
      id: 'region', 
      label: '区域', 
      sortable: true,
      format: (value) => getRegionName(value as string)
    },
    { 
      id: 'count', 
      label: '补机次数', 
      sortable: true,
      align: 'center'
    },
    {
      id: 'timestamp',
      label: '最近补机时间',
      sortable: true,
      format: (value) => value ? new Date(value).toLocaleString('zh-CN') : ''
    }
  ];

  return (
    <>
      <Helmet>
        <title>{`补机历史 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
          补机历史
        </Typography>

        {/* 数据表格 */}
        <Box sx={{ mb: 3 }}>
          <DataTable
            columns={columns}
            data={historyData}
            selectable={false}
          />
        </Box>
      </DashboardContent>
    </>
  );
}