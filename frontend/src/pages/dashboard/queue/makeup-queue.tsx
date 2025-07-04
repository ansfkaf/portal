// src/pages/dashboard/queue/makeup-queue.tsx
import { useEffect, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Paper from '@mui/material/Paper';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import { SvgIcon } from '@mui/material';
import axiosInstance from 'src/lib/axios';
import { DataTable, type Column } from 'src/components/table/data-table';

// 补机队列项类型定义
interface MakeupQueueItem {
  user_id: string;
  region: string;
  region_display: string;
  total_count: number;
  completed_count: number;
  add_time: string;
  status: string;
  remaining: number;
}

// 队列状态摘要类型定义
interface QueueStatus {
  queue_size: number;
  active_tasks: number;
  waiting_tasks: number;
  completed_tasks: number;
  total_machines: number;
  completed_machines: number;
}

// 队列数据类型定义
interface QueueApiResponse {
  code: number;
  message: string;
  data: {
    queue: MakeupQueueItem[];
    status: QueueStatus;
  };
}

// 重置响应接口
interface ResetApiResponse {
  code: number;
  message: string;
  data: {
    message: string;
  };
}

export default function MakeupQueuePage() {
  // 状态管理
  const [queueData, setQueueData] = useState<{
    queue: MakeupQueueItem[];
    status: QueueStatus;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [resetting, setResetting] = useState(false);
  const { addAlert } = useAlert();

  // 获取队列数据
  const fetchQueueData = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.get<QueueApiResponse>('/pool/makeup-queue');
      if (response.data.code === 200) {
        // 按照添加时间倒序排序
        const sortedQueue = [...response.data.data.queue].sort((a, b) => {
          return new Date(b.add_time).getTime() - new Date(a.add_time).getTime();
        });
        
        setQueueData({
          ...response.data.data,
          queue: sortedQueue
        });
      } else {
        addAlert('error', '获取补机队列数据失败');
        setQueueData(null);
      }
    } catch (error) {
      addAlert('error', '获取补机队列数据请求失败');
      setQueueData(null);
    } finally {
      setLoading(false);
    }
  };

  // 初始加载数据
  useEffect(() => {
    fetchQueueData();
  }, []);

  // 重置补机队列
  const handleResetQueue = async () => {
    try {
      setResetting(true);
      const response = await axiosInstance.post<ResetApiResponse>('/pool/reset-makeup');
      
      if (response.data.code === 200) {
        addAlert('success', response.data.data.message || '已重置所有卡住的补机任务');
        // 重新加载数据
        await fetchQueueData();
      } else {
        addAlert('error', '重置补机队列失败');
      }
    } catch (error) {
      addAlert('error', '重置补机队列请求失败');
    } finally {
      setResetting(false);
    }
  };

  // 清空补机队列
  const handleClearQueue = async () => {
    try {
      const response = await axiosInstance.post<ResetApiResponse>('/pool/clear-makeup');
      
      if (response.data.code === 200) {
        addAlert('success', response.data.data.message || '已清空所有补机队列');
        // 重新加载数据
        await fetchQueueData();
      } else {
        addAlert('error', '清空补机队列失败');
      }
    } catch (error) {
      addAlert('error', '清空补机队列请求失败');
    }
  };

  // 获取状态文本颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case '等待中':
        return '#FF9800'; // warning
      case '进行中':
        return '#2196F3'; // info
      case '已完成':
        return '#4CAF50'; // success
      default:
        return '#757575'; // default
    }
  };

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
      sortable: true
    },
    { 
      id: 'region', 
      label: '区域', 
      sortable: true,
      format: (value) => getRegionName(value as string)
    },
    { 
      id: 'total_count', 
      label: '计划补机数', 
      sortable: true,
      align: 'center',
      sortType: 'number'
    },
    { 
      id: 'completed_count', 
      label: '已完成数', 
      sortable: true,
      align: 'center',
      sortType: 'number'
    },
    { 
      id: 'remaining', 
      label: '剩余数量', 
      sortable: true,
      align: 'center',
      sortType: 'number'
    },
    { 
      id: 'status', 
      label: '状态',
      sortable: true,
      align: 'center',
      format: (value) => (
        <Box
          component="span"
          sx={{
            px: 1.5,
            py: 0.5,
            borderRadius: 1,
            bgcolor: `${getStatusColor(value as string)}20`,
            color: getStatusColor(value as string),
            display: 'inline-flex',
            alignItems: 'center',
          }}
        >
          {value}
        </Box>
      )
    },
    { 
      id: 'add_time', 
      label: '添加时间',
      sortable: true,
      sortType: 'date',
      format: (value) => new Date(value as string).toLocaleString('zh-CN')
    }
  ];

  return (
    <>
      <Helmet>
        <title>{`补机队列 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
          补机队列管理
        </Typography>

        {/* 按钮组 */}
        <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
          <CustomButton 
            onClick={handleResetQueue}
            disabled={loading || resetting}
          >
            重置队列
          </CustomButton>
          
          {/* 清空队列按钮 */}
          <CustomButton 
            onClick={handleClearQueue}
          >
            清空队列
          </CustomButton>
        </Stack>

        {/* 状态统计信息 */}
        {queueData && (
          <Box sx={{ mb: 3 }}>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
              <Paper sx={{ p: 2, flex: 1 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>总任务数</Typography>
                <Typography variant="h4">{queueData.status.queue_size}</Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>进行中任务</Typography>
                <Typography variant="h4">{queueData.status.active_tasks}</Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>等待中任务</Typography>
                <Typography variant="h4">{queueData.status.waiting_tasks}</Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>已完成任务</Typography>
                <Typography variant="h4">{queueData.status.completed_tasks}</Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>补机总数</Typography>
                <Typography variant="h4">{queueData.status.total_machines}</Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1 }}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>已完成补机</Typography>
                <Typography variant="h4">{queueData.status.completed_machines}</Typography>
              </Paper>
            </Stack>
          </Box>
        )}

        {/* 补机队列表格 - 使用DataTable替换原来的Table */}
        <Box sx={{ mb: 3 }}>
          <DataTable
            columns={columns}
            data={queueData?.queue || []}
            rowKey="user_id"
            selectable={false}
          />
        </Box>
      </DashboardContent>
    </>
  );
}