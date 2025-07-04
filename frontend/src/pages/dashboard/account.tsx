// src/pages/dashboard/account.tsx
import { useEffect, useState, useMemo } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import { DataTable, type Column } from 'src/components/table/data-table';

// 在这里使用原始的Column类型，让DataTable处理类型转换
// 如果有TypeScript错误但功能正常，可以使用类型断言解决
import { TableSearch } from 'src/components/table/table-search';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import axiosInstance from 'src/lib/axios';
import { Dialog, DialogContent, DialogTitle, IconButton } from '@mui/material';

// 接口类型定义
interface AccountData {
  id: string;
  user_id: string;
  email: string;
  password: string;
  key1: string;
  key2: string;
  quatos: string | null;
  hk: string | null;
  vm_count: number | null; // 修改字段名: hk_count -> vm_count
  region: string | null; // 新增区域字段
  create_time: string;
}

// 区域映射表
const regionMap: Record<string, string> = {
  'ap-east-1': '香港',
  'ap-southeast-1': '新加坡',
  'ap-northeast-3': '日本',
  // 可以根据需要添加更多区域映射
};

interface ApiResponse {
  code: number;
  message: string;
  data: AccountData[];
}

interface HKResponse {
  code: number;
  message: string;
  data: {
    account_id: string;
    status: string;
    message: string;
  }[];
}

interface EC2Response {
  code: number;
  message: string;
  data: {
    account_id: string;
    status: string;
    message: string;
    instances: {
      instance_id: string;
      public_ip: string;
      status: string;
    }[] | null;
  }[];
}

// 清理t3.micro的接口类型
interface CleanT3Response {
  code: number;
  message: string;
  data: {
    account_results: {
      account_id: string;
      status: string;
      message: string;
      found: number;
      deleted: number;
    }[];
    total_found: number;
    total_deleted: number;
    success_count: number;
    fail_count: number;
  };
}

export default function AccountPage() {
  const [accounts, setAccounts] = useState<AccountData[]>([]);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const { addAlert } = useAlert();
  
  // 搜索状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchField, setSearchField] = useState('');

  // 获取账号列表数据
  const fetchAccounts = async () => {
    try {
      const response = await axiosInstance.get<ApiResponse>('/account/list');
      if (response.data.code === 200) {
        const sortedData = response.data.data.sort((a: AccountData, b: AccountData) => 
          parseInt(a.id) - parseInt(b.id)
        );
        setAccounts(sortedData);
      } else {
        addAlert('error', '获取数据失败');
      }
    } catch (error) {
      addAlert('error', '获取账号列表失败');
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, []);

  // 检测账号
  const handleCheck = async () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要检测的账号');
      return;
    }

    try {
      const response = await axiosInstance.post('/account/check', {
        account_ids: selectedIds
      });
      
      if (response.data.code === 200) {
        addAlert('success', '检测成功');
        fetchAccounts(); // 重新加载数据
      } else {
        addAlert('error', '检测失败');
      }
    } catch (error) {
      addAlert('error', '检测请求失败');
    }
  };

  // 开机功能
  const handlePowerOn = async () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要开机的账号');
      return;
    }

    try {
      const response = await axiosInstance.post<EC2Response>('/account/create-instance', {
        account_ids: selectedIds
      });
      
      if (response.data.code === 200) {
        const results = response.data.data;
        const successCount = results.filter(r => r.status === '成功').length;
        const failedCount = results.filter(r => r.status === '失败').length;

        let message = '';
        if (successCount > 0) {
          message += `${successCount}个账号开机成功`;
        }
        if (failedCount > 0) {
          message += failedCount > 0 && successCount > 0 ? '，' : '';
          message += `${failedCount}个账号开机失败`;
        }

        if (successCount > 0) {
          addAlert('success', message);
        } else {
          addAlert('error', message);
        }
        
        fetchAccounts(); // 重新加载数据
      }
    } catch (error) {
      addAlert('error', '开机请求失败');
    }
  };

  // 申请HK区
  const handleApplyHK = async () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要申请HK区的账号');
      return;
    }

    try {
      const response = await axiosInstance.post<HKResponse>('/account/apply-hk', {
        account_ids: selectedIds
      });
      
      if (response.data.code === 200) {
        const results = response.data.data;
        const successCount = results.filter(r => r.status === '成功').length;
        const failedCount = results.filter(r => r.status === '失败').length;
        const failedMessages = results
          .filter(r => r.status === '失败')
          .map(r => r.message)
          .join('、');

        if (successCount > 0) {
          addAlert('success', `${successCount}个账号香港区域已启用`);
        }
        if (failedCount > 0) {
          addAlert('error', `${failedCount}个账号失败：${failedMessages}`);
        }
        
        fetchAccounts(); // 重新加载数据
      }
    } catch (error) {
      addAlert('error', '申请HK区请求失败');
    }
  };

  // 清理t3.micro实例
  const handleCleanT3Micro = async () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要清理t3.micro的账号');
      return;
    }

    try {
      const response = await axiosInstance.post<CleanT3Response>('/account/clean-t3-micro', {
        account_ids: selectedIds
      });
      
      if (response.data.code === 200) {
        const { success_count, fail_count, total_found, total_deleted, account_results } = response.data.data;
        
        // 成功消息
        if (success_count > 0) {
          let successMessage = `操作成功：${success_count}`;
          if (total_found > 0) {
            successMessage += `，共发现${total_found}个t3.micro实例，已清理${total_deleted}个`;
          } else {
            successMessage += '，未发现t3.micro实例';
          }
          addAlert('success', successMessage);
        }
        
        // 失败消息
        if (fail_count > 0) {
          const failedMessages = account_results
            .filter(r => r.status === '失败')
            .map(r => r.message)
            .join('、');
          
          addAlert('error', `${fail_count}个账号清理失败：${failedMessages}`);
        }
        
        fetchAccounts(); // 重新加载数据
      } else {
        addAlert('error', '清理t3.micro失败');
      }
    } catch (error) {
      addAlert('error', '清理t3.micro请求失败');
    }
  };

  // 查看账号详情
  const handleViewDetails = () => {
    setIsDialogOpen(true);
  };

  // 删除账号
  const handleDelete = async () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要删除的账号');
      return;
    }

    try {
      const response = await axiosInstance.post('/account/delete', {
        account_ids: selectedIds
      });
      
      if (response.data.code === 200) {
        addAlert('success', '删除成功');
        setSelectedIds([]); // 清空选择
        fetchAccounts(); // 重新加载数据
      } else {
        addAlert('error', '删除失败');
      }
    } catch (error) {
      addAlert('error', '删除请求失败');
    }
  };
  
  // 处理搜索
  const handleSearch = (keyword: string, field: string) => {
    setSearchKeyword(keyword);
    setSearchField(field);
  };
  
  // 过滤账号数据，为非香港区域的账号清空hk字段
  const processedAccounts = useMemo(() => {
    return accounts.map(account => {
      // 创建账号数据的拷贝
      const processedAccount = { ...account };
      
      // 如果不是香港区域，清空hk字段
      if (processedAccount.region !== 'ap-east-1') {
        processedAccount.hk = '';
      } else if (!processedAccount.hk) {
        // 如果是香港区域且hk字段为空，设置为"未启用"
        processedAccount.hk = '未启用';
      }
      
      return processedAccount;
    });
  }, [accounts]);
  
  // 根据搜索条件过滤账号
  const filteredAccounts = useMemo(() => {
    if (!searchKeyword || !searchField) return processedAccounts;
    
    return processedAccounts.filter(account => {
      const value = account[searchField as keyof AccountData];
      if (value === null || value === undefined) return false;
      
      return String(value).toLowerCase().includes(searchKeyword.toLowerCase());
    });
  }, [processedAccounts, searchKeyword, searchField]);

  // 主表格列定义
  const columns: Column[] = [
    { 
      id: 'id', 
      label: '账号ID', 
      sortable: true,
      width: '100px'
    },
    { 
      id: 'quatos', 
      label: '配额', 
      sortable: true
    },
    { 
      id: 'region', 
      label: '区域',
      sortable: true,
      format: (value) => value ? regionMap[value] || value : '未设置'
    },
    { 
      id: 'hk', 
      label: 'HK区状态',
      sortable: true,
      format: (value) => {
        // 由于无法直接访问row参数，我们使用value本身来判断
        // 只要有值，就显示该值（通常是"启用"）
        // 对于非香港区域，前端在展示时会将该字段置空，所以这里直接返回值本身
        return value || '';
      }
    },
    { 
      id: 'vm_count',
      label: '实例数',
      sortable: true,
      align: 'center',
      format: (value) => value ?? 0
    },
    {
      id: 'create_time',
      label: '添加时间',
      sortable: true,
      format: (value) => new Date(value).toLocaleString('zh-CN')
    }
  ];

  // 账号详情对话框列定义
  const detailColumns: Column[] = [
    { 
      id: 'id', 
      label: '账号ID', 
      width: '100px'
    },
    { 
      id: 'email', 
      label: '账号'
    },
    { 
      id: 'password', 
      label: '密码'
    },
    { 
      id: 'key1', 
      label: 'Key1'
    },
    { 
      id: 'key2', 
      label: 'Key2'
    },
    { 
      id: 'region', 
      label: '区域',
      // @ts-ignore
      format: (value) => value ? regionMap[value] || value : '未设置'
    },
    { 
      id: 'create_time', 
      label: '创建时间',
      format: (value) => new Date(value).toLocaleString('zh-CN')
    }
  ];

  return (
    <>
      <Helmet>
        <title>{`账号管理 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
          账号管理
        </Typography>

        {/* 按钮组和搜索框在同一行 */}
        <Box sx={{ 
          display: 'flex', 
          flexDirection: { xs: 'column', md: 'row' }, 
          justifyContent: 'space-between',
          alignItems: { xs: 'stretch', md: 'center' },
          mb: 3 
        }}>
          {/* 按钮组 */}
          <Stack 
            direction="row" 
            spacing={2} 
            sx={{ 
              mb: { xs: 2, md: 0 },
              flexWrap: 'wrap',
              gap: 1
            }}
          >
            <CustomButton onClick={handleCheck}>
              检测
            </CustomButton>
            <CustomButton onClick={handlePowerOn}>
              开机
            </CustomButton>
            <CustomButton onClick={handleApplyHK}>
              申请HK区
            </CustomButton>
            <CustomButton onClick={handleCleanT3Micro}>
              清理t3
            </CustomButton>
            <CustomButton onClick={handleViewDetails}>
              查看账号
            </CustomButton>
            <CustomButton onClick={handleDelete}>
              删除
            </CustomButton>
          </Stack>
          
          {/* 搜索框 - 更新列定义 */}
          <TableSearch
            columns={[
              { id: 'id', label: '账号ID' },
              { id: 'quatos', label: '配额' },
              { id: 'region', label: '区域' },
              { id: 'hk', label: 'HK区状态' },
              { id: 'vm_count', label: '实例数' }
            ]}
            onSearch={handleSearch}
            position="right"
            width={300}
            defaultField="id"
          />
        </Box>

        {/* 主数据表格 */}
        <Box sx={{ mb: 3 }}>
          <DataTable
            columns={columns}
            data={filteredAccounts} // 使用过滤后的数据
            selectable
            onSelectionChange={setSelectedIds}
            searchable={false} // 关闭表格内部搜索功能
          />
        </Box>

        {/* 账号详情对话框 */}
        <Dialog
          open={isDialogOpen}
          onClose={() => setIsDialogOpen(false)}
          maxWidth="lg"
          fullWidth
        >
          <DialogTitle sx={{ m: 0, p: 2 }}>
            账号详情
            <IconButton
              onClick={() => setIsDialogOpen(false)}
              sx={{
                position: 'absolute',
                right: 8,
                top: 8,
                color: 'grey.500'
              }}
            >
              ×
            </IconButton>
          </DialogTitle>
          <DialogContent>
            <DataTable
              columns={detailColumns}
              data={accounts}
              selectable={false}
            />
          </DialogContent>
        </Dialog>
      </DashboardContent>
    </>
  );
}