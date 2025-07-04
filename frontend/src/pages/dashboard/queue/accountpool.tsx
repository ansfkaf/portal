// src/pages/dashboard/accountpool.tsx
import { useEffect, useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import { DataTable, type Column } from 'src/components/table/data-table';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import axiosInstance from 'src/lib/axios';
import Tooltip from '@mui/material/Tooltip';
import Chip from '@mui/material/Chip';

// 账号数据接口定义
interface AccountData {
  id: string;
  user_id: string;
  key1: string;
  key2: string;
  email: string;
  password: string;
  region: string; // 新增区域字段
  create_time: string;
  is_skipped: boolean;
  error_note: string;
  // 跳过的实例类型
  skipped_instance_types: Record<string, boolean> | null;
  // 区域使用计数统计
  region_used_count: number;
}

// 区域映射表
const regionMap: Record<string, string> = {
  'ap-east-1': '香港',
  'ap-southeast-1': '新加坡',
  'ap-northeast-3': '日本',
};

// API响应接口
interface PoolApiResponse {
  code: number;
  message: string;
  data: {
    total: number;
    available: number;
    accounts: AccountData[];
  };
}

// 重置账号响应接口
interface ResetApiResponse {
  code: number;
  message: string;
  data: {
    failed: {
      count: number;
      ids: string[];
    };
    success: {
      count: number;
      ids: string[];
    };
  };
}

// 通用API响应接口
interface ApiResponse {
  code: number;
  message: string;
  data: any;
}

export default function AccountPoolPage() {
  // 状态管理
  const [accounts, setAccounts] = useState<AccountData[]>([]);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const { addAlert } = useAlert();

  // 获取账号池数据
  const fetchAccountPool = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.get<PoolApiResponse>('/pool/accountpool');
      if (response.data.code === 200) {
        // 确保accounts始终是数组，即使API返回null或undefined
        setAccounts(response.data.data.accounts || []);
      } else {
        addAlert('error', '获取账号池数据失败');
        // 设置为空数组而不是null
        setAccounts([]);
      }
    } catch (error) {
      addAlert('error', '获取账号池数据请求失败');
      // 出错时也确保设置为空数组
      setAccounts([]);
    } finally {
      setLoading(false);
    }
  };

  // 初始加载数据
  useEffect(() => {
    fetchAccountPool();
  }, []);

  // 重置账号
  const handleResetAccounts = async () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要重置的账号');
      return;
    }

    try {
      setLoading(true);
      const response = await axiosInstance.post<ResetApiResponse>('/pool/reset-accounts', {
        account_ids: selectedIds
      });
      
      if (response.data.code === 200) {
        const { success, failed } = response.data.data;
        
        if (success.count > 0) {
          addAlert('success', `成功重置 ${success.count} 个账号`);
        }
        
        if (failed.count > 0) {
          addAlert('warning', `${failed.count} 个账号重置失败`);
        }
        
        // 重新加载数据
        fetchAccountPool();
        // 清空选择
        setSelectedIds([]);
      } else {
        addAlert('error', '重置账号失败');
      }
    } catch (error) {
      addAlert('error', '重置账号请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 清空补机历史和冷却状态
  const handleClearHistory = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.post<ApiResponse>('/monitor/admin/clear');
      
      if (response.data.code === 200) {
        addAlert('success', response.data.data.message || '已清空所有区域的补机历史记录并重置账号冷却状态');
        // 刷新数据
        fetchAccountPool();
      } else {
        addAlert('error', '清理队列失败');
      }
    } catch (error) {
      addAlert('error', '清理队列请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 渲染跳过的实例类型
  const renderSkippedInstanceTypes = (value: any): React.ReactNode => {
    const skippedTypes = value as Record<string, boolean> | null;
    if (!skippedTypes || Object.keys(skippedTypes).length === 0) {
      return '无';
    }

    const types = Object.keys(skippedTypes).filter(type => skippedTypes[type]);
    return (
      <Stack direction="row" spacing={1} flexWrap="wrap">
        {types.map(type => (
          <Chip 
            key={type} 
            label={type} 
            size="small" 
            color="error" 
            sx={{ my: 0.5 }}
          />
        ))}
      </Stack>
    );
  };

  // 表格列定义
  const columns: Column[] = [
    { 
      id: 'id', 
      label: '账号ID', 
      sortable: true,
      width: '80px'
    },
    { 
      id: 'region', 
      label: '区域', 
      sortable: true,
      format: (value) => value ? regionMap[value] || value : '未设置'
    },
    { 
      id: 'region_used_count', 
      label: '已使用实例数',
      sortable: true,
      align: 'center'
    },
    { 
      id: 'is_skipped', 
      label: '是否跳过',
      sortable: true,
      align: 'center',
      format: (value) => value ? '是' : '否'
    },
    { 
      id: 'error_note', 
      label: '错误信息',
      sortable: true,
      // 使用Tooltip组件包装错误信息，以便在内容过长时能够显示完整内容
      format: (value) => value ? (
        <Tooltip title={value} arrow>
          <Typography 
            variant="body2" 
            sx={{ 
              maxWidth: '200px',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap'
            }}
          >
            {value}
          </Typography>
        </Tooltip>
      ) : ''
    },
    {
      id: 'skipped_instance_types',
      label: '跳过的实例类型',
      sortable: false,
      format: (value) => renderSkippedInstanceTypes(value)
    },
    {
      id: 'create_time',
      label: '添加时间',
      sortable: true,
      format: (value) => value ? new Date(value).toLocaleString('zh-CN') : ''
    }
  ];

  return (
    <>
      <Helmet>
        <title>{`账号池队列 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
          账号池队列
        </Typography>

        {/* 按钮组 */}
        <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
          <CustomButton 
            onClick={handleResetAccounts}
            disabled={loading || accounts.length === 0}
          >
            重置账号
          </CustomButton>
          <CustomButton 
            onClick={handleClearHistory}
            disabled={loading}
            color="warning"
          >
            清理队列
          </CustomButton>
        </Stack>

        {/* 数据表格 */}
        <Box sx={{ mb: 3 }}>
          <DataTable
            columns={columns}
            data={accounts} // 确保accounts始终是数组
            selectable={accounts.length > 0} // 只有有数据时才可选
            onSelectionChange={setSelectedIds}
          />
        </Box>
      </DashboardContent>
    </>
  );
}