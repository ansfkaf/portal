// src/pages/dashboard/instance.tsx
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
import { FormControl, InputLabel, MenuItem, Select } from '@mui/material';

// 接口类型定义
interface AccountItem {
  id: string;
  user_id: string;
  key1: string;
  key2: string;
  email: string | null;
  password: string | null;
  quatos: string | null;
  hk: string | null;
  vm_count: number | null; // 与账号管理页保持一致，由hk_count更新为vm_count
  region: string | null; // 新增区域字段
  create_time: string | null;
}

interface Instance {
  instance_id: string;
  public_ip: string;
  instance_type: string;
  state: string;
  launch_time: string;
  account_id: string;
  id?: string; // 为DataTable组件添加id字段
}

interface InstanceResponse {
  account_id: string;
  instances: Instance[] | null;
}

interface ActionResponse {
  account_id: string;
  instance_id: string;
  status: string;
  message: string;
  old_ip?: string;
  new_ip?: string;
}

interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// 区域映射
const regionMap = {
  'ap-east-1': '香港',
  'ap-northeast-3': '日本',
  'ap-southeast-1': '新加坡'
};

// 区域选项
const regionOptions = [
  { label: '香港', value: 'ap-east-1' },
  { label: '日本', value: 'ap-northeast-3' },
  { label: '新加坡', value: 'ap-southeast-1' }
];

export default function InstancePage() {
  const [accounts, setAccounts] = useState<AccountItem[]>([]);
  const [selectedAccountId, setSelectedAccountId] = useState<string>('');
  const [selectedRegion, setSelectedRegion] = useState<string>('ap-east-1');
  const [instances, setInstances] = useState<Instance[]>([]);
  const [selectedInstanceIds, setSelectedInstanceIds] = useState<string[]>([]);
  const [isChangingIp, setIsChangingIp] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [customMessage, setCustomMessage] = useState<string>('');
  const { addAlert } = useAlert();

  // 获取可用账号列表
  const fetchAccounts = async () => {
    try {
      const response = await axiosInstance.get<ApiResponse<AccountItem[]>>('/instance/account_list');
      if (response.data.code === 200) {
        setAccounts(response.data.data);
        if (response.data.data.length > 0) {
          const firstAccount = response.data.data[0];
          setSelectedAccountId(firstAccount.id);
          
          // 设置默认区域为所选账号的区域
          if (firstAccount.region) {
            setSelectedRegion(firstAccount.region);
          }
        }
      } else {
        addAlert('error', '获取账号列表失败');
      }
    } catch (error) {
      addAlert('error', '获取账号列表失败');
    }
  };

  // 查询实例列表
  const fetchInstances = async () => {
    if (!selectedAccountId) {
      addAlert('warning', '请选择账号');
      return;
    }

    try {
      const response = await axiosInstance.post<ApiResponse<InstanceResponse[]>>('/instance/list', {
        account_ids: [selectedAccountId],
        region: selectedRegion
      });

      if (response.data.code === 200) {
        // 处理返回的数据
        const responseData = response.data.data;
        const accountData = responseData[0]; // 因为我们只查询一个账号，所以取第一个

        if (!accountData.instances) {
          // 账号存在但没有实例的情况
          setInstances([]);
          return;
        }

        // 有实例数据的情况
        const allInstances = accountData.instances.map(instance => ({
          ...instance,
          id: instance.instance_id,
          account_id: accountData.account_id
        }));
        
        setInstances(allInstances);
        setCustomMessage('');
      } else {
        addAlert('error', '获取实例列表失败');
      }
    } catch (error) {
      addAlert('error', '查询实例失败');
    }
  };

  // 处理账号选择变化
  const handleAccountChange = (event: any) => {
    const newAccountId = event.target.value;
    setSelectedAccountId(newAccountId);
    
    // 更新区域为所选账号的默认区域
    const selectedAccount = accounts.find(account => account.id === newAccountId);
    if (selectedAccount && selectedAccount.region) {
      setSelectedRegion(selectedAccount.region);
    }
    
    setInstances([]); // 清空实例列表
    setSelectedInstanceIds([]); // 清空选择状态
    setCustomMessage(''); // 清空自定义消息
  };

  // 处理区域选择变化
  const handleRegionChange = (event: any) => {
    setSelectedRegion(event.target.value);
    setInstances([]); // 清空实例列表
    setSelectedInstanceIds([]); // 清空选择状态
    setCustomMessage(''); // 清空自定义消息
  };

  // 更换IP
  const handleChangeIP = async () => {
    if (selectedInstanceIds.length === 0) {
      addAlert('warning', '请选择要更换IP的实例');
      return;
    }

    // 立即显示操作开始的提示
    addAlert('info', `正在更换 ${selectedInstanceIds.length} 个实例的IP，请稍候...`);
    setIsChangingIp(true);

    // 构建请求数据
    const instancesForUpdate = selectedInstanceIds.map(instanceId => {
      const instance = instances.find(i => i.instance_id === instanceId);
      if (!instance) {
        throw new Error(`找不到实例ID: ${instanceId}`);
      }
      return {
        account_id: instance.account_id,
        instance_id: instance.instance_id,
        region: selectedRegion // 添加区域参数
      };
    });

    try {
      const response = await axiosInstance.post<ApiResponse<ActionResponse[]>>('/instance/change-ip', {
        instances: instancesForUpdate
      });
      
      if (response.data.code === 200) {
        const results = response.data.data;
        const successCount = results.filter(r => r.status === '成功').length;
        const failedResults = results.filter(r => r.status === '失败');
        const failedCount = failedResults.length;

        if (successCount > 0) {
          addAlert('success', `更换IP成功：${successCount}`);
        }
        
        if (failedCount > 0) {
          const failedMessages = failedResults.map(r => r.message).join('、');
          addAlert('error', `更换IP失败：${failedCount}，${failedMessages}`);
        }

        // 重新获取实例列表，但保持选择状态
        const currentSelected = [...selectedInstanceIds];
        await fetchInstances();
        setSelectedInstanceIds(currentSelected);
      }
    } catch (error) {
      addAlert('error', '更换IP请求失败');
    } finally {
      setIsChangingIp(false);
    }
  };

  // 删除实例
  const handleDelete = async () => {
    if (selectedInstanceIds.length === 0) {
      addAlert('warning', '请选择要删除的实例');
      return;
    }

    // 立即显示操作开始的提示
    addAlert('info', `正在删除 ${selectedInstanceIds.length} 个实例，请稍候...`);
    setIsDeleting(true);

    // 构建请求数据
    const instancesForDelete = selectedInstanceIds.map(instanceId => {
      const instance = instances.find(i => i.instance_id === instanceId);
      if (!instance) {
        throw new Error(`找不到实例ID: ${instanceId}`);
      }
      return {
        account_id: instance.account_id,
        instance_id: instance.instance_id,
        region: selectedRegion // 添加区域参数
      };
    });

    try {
      const response = await axiosInstance.post<ApiResponse<ActionResponse[]>>('/instance/delete', {
        instances: instancesForDelete
      });
      
      if (response.data.code === 200) {
        const results = response.data.data;
        const successCount = results.filter(r => r.status === '成功').length;
        const failedResults = results.filter(r => r.status === '失败');
        const failedCount = failedResults.length;

        if (successCount > 0) {
          addAlert('success', `删除成功：${successCount}`);
        }
        
        if (failedCount > 0) {
          const failedMessages = failedResults.map(r => r.message).join('、');
          addAlert('error', `删除失败：${failedCount}，${failedMessages}`);
        }

        await fetchInstances();
        setSelectedInstanceIds([]);
      }
    } catch (error) {
      addAlert('error', '删除请求失败');
    } finally {
      setIsDeleting(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, []);

  // 表格列定义
  const columns: Column[] = [
    {
      id: 'account_id',
      label: '账号ID',
      width: '100px'
    },
    {
      id: 'instance_id',
      label: '实例ID',
      width: '200px'
    },
    {
      id: 'public_ip',
      label: 'IP地址'
    },
    {
      id: 'instance_type',
      label: '实例类型'
    },
    {
      id: 'state',
      label: '状态'
    },
    {
      id: 'launch_time',
      label: '启动时间',
      format: (value) => new Date(value).toLocaleString('zh-CN')
    }
  ];

  // 选择变更处理
  const handleSelectionChange = (selectedIds: string[]) => {
    // 验证选择的ID是否存在于当前实例列表中
    const validIds = selectedIds.filter(id => 
      instances.some(instance => instance.instance_id === id)
    );
    setSelectedInstanceIds(validIds);
  };

  // 获取区域名称
  const getRegionName = (code: string) => {
    return regionMap[code as keyof typeof regionMap] || '未知区域';
  };

  return (
    <>
      <Helmet>
        <title>{`实例管理 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
          实例管理
        </Typography>

        {/* 查询条件 */}
        <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
          <FormControl sx={{ minWidth: 200 }}>
            <InputLabel>账户ID</InputLabel>
            <Select
              value={selectedAccountId}
              onChange={handleAccountChange}
              label="账户ID"
            >
              {accounts.map((account) => (
                <MenuItem key={account.id} value={account.id}>
                  {account.id} - {account.region ? getRegionName(account.region) : '未知区域'}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <FormControl sx={{ minWidth: 120 }}>
            <InputLabel>区域</InputLabel>
            <Select
              value={selectedRegion}
              onChange={handleRegionChange}
              label="区域"
            >
              {regionOptions.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <CustomButton onClick={fetchInstances}>
            查询
          </CustomButton>
          <CustomButton 
            onClick={handleDelete}
            disabled={isDeleting}
          >
            {isDeleting ? '删除中...' : '删除'}
          </CustomButton>
          <CustomButton 
            onClick={handleChangeIP}
            disabled={isChangingIp}
          >
            {isChangingIp ? '更换IP中...' : '更换IP'}
          </CustomButton>
        </Stack>

        {/* 数据表格 */}
        <Box sx={{ mb: 3 }}>
          <DataTable
            columns={columns}
            data={instances}
            selectable
            rowKey="instance_id"
            onSelectionChange={handleSelectionChange}
          />
        </Box>
      </DashboardContent>
    </>
  );
}