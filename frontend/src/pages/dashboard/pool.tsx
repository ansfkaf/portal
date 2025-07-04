// src/pages/dashboard/pool.tsx
import { useEffect, useState, useRef, useCallback, useMemo } from 'react';
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
import { useIsAdmin } from 'src/auth/hooks/use-is-admin';
import { AdminModeToggle, useAdminMode } from 'src/components/admin-mode-toggle';
import { TableSearch } from 'src/components/table/table-search';

// 接口类型定义
interface PoolInstance {
  instance_id: string;
  instance_type: string;
  user_id: string;
  account_id: string;
  ipv4: string;
  region: string;
  launch_time: string;
  report_time: string;
}

interface PoolResponse {
  list: PoolInstance[];
  total: number;
}

interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

interface ActionResponse {
  account_id: string;
  instance_id: string;
  status: string;
  message: string;
  old_ip?: string;
  new_ip?: string;
}

export default function PoolPage() {
  const [instances, setInstances] = useState<PoolInstance[]>([]);
  const [selectedInstanceIds, setSelectedInstanceIds] = useState<string[]>([]);
  const [isChangingIp, setIsChangingIp] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true);
  const [searchKeyword, setSearchKeyword] = useState('');
  const { isAdminMode } = useAdminMode();
  const [searchField, setSearchField] = useState('account_id');
  
  const { addAlert } = useAlert();
  const isAdmin = useIsAdmin(); // 获取当前用户是否为管理员
  
  // 用于跟踪自动刷新计时器
  const timerRef = useRef<NodeJS.Timeout | null>(null);
  // 用于跟踪自动刷新开始时间
  const startTimeRef = useRef<number>(Date.now());
  // 用于跟踪管理员模式的上一次状态，避免重复提示
  const prevAdminModeRef = useRef<boolean | null>(null);

  // 数据排序函数
  const sortInstancesData = (data: PoolInstance[]): PoolInstance[] => {
    return [...data].sort((a, b) => {
      // 首先按账号ID排序（转换为数字比较）
      const accountIdA = parseInt(a.account_id, 10);
      const accountIdB = parseInt(b.account_id, 10);
      
      if (!isNaN(accountIdA) && !isNaN(accountIdB)) {
        const accountCompare = accountIdA - accountIdB;
        if (accountCompare !== 0) return accountCompare;
      } else {
        // 如果转换数字失败，用字符串比较
        const accountCompare = a.account_id.localeCompare(b.account_id);
        if (accountCompare !== 0) return accountCompare;
      }
      
      // 如果是管理员模式，按用户ID排序（转换为数字比较）
      if (isAdminMode) {
        const userIdA = parseInt(a.user_id, 10);
        const userIdB = parseInt(b.user_id, 10);
        
        if (!isNaN(userIdA) && !isNaN(userIdB)) {
          const userCompare = userIdA - userIdB;
          if (userCompare !== 0) return userCompare;
        } else {
          // 如果转换数字失败，用字符串比较
          const userCompare = a.user_id.localeCompare(b.user_id);
          if (userCompare !== 0) return userCompare;
        }
      }
      
      // 最后按实例ID排序
      return a.instance_id.localeCompare(b.instance_id);
    });
  };

  // 查询实例列表 - 使用 useCallback 避免不必要的重渲染
  const fetchInstances = useCallback(async () => {
    try {
      // 修改：确保同时检查用户是否为管理员
      const endpoint = isAdmin && isAdminMode ? '/pool/admin' : '/pool';
      const response = await axiosInstance.post<ApiResponse<PoolResponse>>(endpoint);
      
      if (response.data.code === 200) {
        // 对获取的数据进行排序
        const sortedData = sortInstancesData(response.data.data.list);
        setInstances(sortedData);
      } else {
        addAlert('error', '获取在线实例列表失败');
      }
    } catch (error) {
      addAlert('error', '查询在线实例失败');
      // 出错时停止自动刷新
      setAutoRefreshEnabled(false);
    }
  }, [isAdmin, isAdminMode, addAlert]);
  
  // 初始加载和模式变化时获取数据
  useEffect(() => {
    fetchInstances();
    // 模式变化时清空已选择的实例
    setSelectedInstanceIds([]);
    
    // 重置自动刷新计时
    startTimeRef.current = Date.now();
    
    // 确保自动刷新开启
    setAutoRefreshEnabled(true);
    
    // 清理旧的计时器
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    
    // 检查是否是第一次加载或模式真的发生了变化
    if (prevAdminModeRef.current !== null && prevAdminModeRef.current !== isAdminMode) {
      // 只有当模式真的变化时才显示提示
      addAlert('info', isAdminMode ? '已切换至管理员模式' : '已切换至普通模式');
    }
    
    // 更新上一次的模式状态
    prevAdminModeRef.current = isAdminMode;
    
    // 日志输出，帮助调试
    console.log('管理员模式状态变化:', isAdminMode ? '管理员模式' : '普通模式');
  }, [isAdminMode, fetchInstances, addAlert]);

  // 自动刷新控制
  useEffect(() => {
    if (autoRefreshEnabled) {
      // 设置定时器
      timerRef.current = setInterval(() => {
        // 检查是否超过5分钟
        const currentTime = Date.now();
        const elapsedMinutes = (currentTime - startTimeRef.current) / (1000 * 60);
        
        if (elapsedMinutes >= 5) {
          // 超过5分钟，停止自动刷新
          setAutoRefreshEnabled(false);
          if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
          }
          addAlert('info', '自动刷新已停止，已达到5分钟限制');
        } else {
          // 未超过5分钟，继续刷新
          fetchInstances();
        }
      }, 5000); // 每5秒刷新一次
    } else if (timerRef.current) {
      // 如果autoRefreshEnabled为false且定时器存在，清除定时器
      clearInterval(timerRef.current);
      timerRef.current = null;
    }

    // 清理函数
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [autoRefreshEnabled, fetchInstances]); // 依赖fetchInstances而不是isAdminMode

  // 在这里添加新的 useEffect 钩子
  useEffect(() => {
    // 强制设置搜索字段，确保模式切换时立即生效
    if (isAdminMode) {
      setSearchField('user_id');
    } else {
      setSearchField('account_id');
    }
  }, [isAdminMode]);
  
  // 手动刷新
  const handleRefresh = () => {
    fetchInstances();
    // 重置自动刷新计时
    startTimeRef.current = Date.now();
    setAutoRefreshEnabled(true);
    addAlert('success', '数据已刷新，自动刷新已重新启动');
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
        region: instance.region  // 添加区域信息
      };
    });
  
    try {
      const response = await axiosInstance.post<ApiResponse<ActionResponse[]>>('/pool/change-ip', {
        instances: instancesForUpdate
      });
      
      if (response.data.code === 200) {
        const results = response.data.data;
        const successCount = results.filter(r => r.status === '成功').length;
        const failedResults = results.filter(r => r.status === '失败');
        const failedCount = failedResults.length;

        // 本地更新IP地址，不再重新获取数据
        if (successCount > 0) {
          setInstances(prevInstances => {
            const updatedInstances = prevInstances.map(instance => {
              // 查找是否有对应的更新结果
              const result = results.find(r => 
                r.instance_id === instance.instance_id && 
                r.account_id === instance.account_id && 
                r.status === '成功'
              );
              
              // 如果找到并且有新IP，则更新
              if (result && result.new_ip) {
                return { ...instance, ipv4: result.new_ip };
              }
              return instance;
            });
            
            // 确保数据排序一致
            return sortInstancesData(updatedInstances);
          });
          
          addAlert('success', `更换IP成功：${successCount}`);
        }
        
        if (failedCount > 0) {
          const failedMessages = failedResults.map(r => r.message).join('、');
          addAlert('error', `更换IP失败：${failedCount}，${failedMessages}`);
        }
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
          region: instance.region  // 添加区域信息
        };
      });

      try {
        const response = await axiosInstance.post<ApiResponse<ActionResponse[]>>('/pool/delete', {
          instances: instancesForDelete
        });
      
      if (response.data.code === 200) {
        const results = response.data.data;
        const successCount = results.filter(r => r.status === '成功').length;
        const failedResults = results.filter(r => r.status === '失败');
        const failedCount = failedResults.length;

        if (successCount > 0) {
          // 本地删除成功删除的实例
          const successfullyDeletedIds = results
            .filter(r => r.status === '成功')
            .map(r => r.instance_id);
            
          // 从本地数据中删除这些实例
          setInstances(prevInstances => 
            prevInstances.filter(instance => 
              !successfullyDeletedIds.includes(instance.instance_id)
            )
          );
          
          // 同时从已选择列表中移除这些ID
          setSelectedInstanceIds(prevSelectedIds => 
            prevSelectedIds.filter(id => !successfullyDeletedIds.includes(id))
          );
          
          addAlert('success', `删除成功：${successCount}`);
        }
        
        if (failedCount > 0) {
          const failedMessages = failedResults.map(r => r.message).join('、');
          addAlert('error', `删除失败：${failedCount}，${failedMessages}`);
        }
      }
    } catch (error) {
      addAlert('error', '删除请求失败');
    } finally {
      setIsDeleting(false);
    }
  };
  // 处理搜索
    const handleSearch = (keyword: string, field: string) => {
      setSearchKeyword(keyword);
      setSearchField(field);
    };

    // 根据搜索条件过滤实例
    const filteredInstances = useMemo(() => {
      if (!searchKeyword || !searchField) return instances;
      
      return instances.filter(instance => {
        const value = instance[searchField as keyof PoolInstance];
        if (value === null || value === undefined) return false;
        
        return String(value).toLowerCase().includes(searchKeyword.toLowerCase());
      });
    }, [instances, searchKeyword, searchField]);

  // 根据当前模式获取表格列
  const getColumns = (): Column[] => {
    const baseColumns: Column[] = [
      {
        id: 'account_id',
        label: '账号ID',
        width: '100px',
        sortable: true,
        sortType: 'numeric-string' // 添加此属性
      },
      {
        id: 'instance_id',
        label: '实例ID',
        width: '200px',
        sortable: true
      },
      {
        id: 'ipv4',
        label: 'IP地址',
        sortable: true
      },
      {
        id: 'instance_type',
        label: '实例类型',
        sortable: true
      },
      {
        id: 'region',
        label: '区域',
        sortable: true
      },
      {
        id: 'launch_time',
        label: '启动时间',
        sortable: true,
        format: (value) => new Date(value).toLocaleString('zh-CN')
      },
      {
        id: 'report_time',
        label: '上报时间',
        sortable: true,
        format: (value) => new Date(value).toLocaleString('zh-CN')
      }
    ];

    // 在管理员模式下添加用户ID列
    if (isAdminMode) {
      // 在账号ID列后插入用户ID列
      baseColumns.splice(1, 0, {
        id: 'user_id',
        label: '用户ID',
        width: '100px',
        sortable: true,
        sortType: 'numeric-string' // 添加此属性
      });
    }

    return baseColumns;
  };

  // 选择变更处理
  const handleSelectionChange = (selectedIds: string[]) => {
    const validIds = selectedIds.filter(id => 
      instances.some(instance => instance.instance_id === id)
    );
    setSelectedInstanceIds(validIds);
  };

  return (
    <>
      <Helmet>
        <title>{`在线实例 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Stack 
          direction="row" 
          alignItems="center" 
          justifyContent="space-between" 
          sx={{ mb: { xs: 3, md: 5 } }}
        >
          <Typography variant="h4">
            在线实例
          </Typography>
          
          {/* 使用新的AdminModeToggle组件 */}
          <AdminModeToggle 
            visible={isAdmin}
            // 移除了onModeChange属性，不再需要它
          />
        </Stack>

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
            <CustomButton onClick={handleRefresh}>
              刷新
            </CustomButton>
            <CustomButton 
              onClick={handleChangeIP}
              disabled={isChangingIp}
            >
              {isChangingIp ? '更换IP中...' : '更换IP'}
            </CustomButton>
            <CustomButton 
              onClick={handleDelete}
              disabled={isDeleting}
            >
              {isDeleting ? '删除中...' : '删除'}
            </CustomButton>
          </Stack>
          
          {/* 搜索框 */}
          <TableSearch
            key={isAdminMode ? 'admin-search' : 'user-search'} // 添加 key 属性
            columns={[
              ...(isAdminMode ? [{ id: 'user_id', label: '用户ID' }] : []),
              { id: 'account_id', label: '账号ID' },
              { id: 'instance_id', label: '实例ID' },
              { id: 'ipv4', label: 'IP地址' },
              { id: 'instance_type', label: '实例类型' },
              { id: 'region', label: '区域' },
              { id: 'launch_time', label: '启动时间' },
              { id: 'report_time', label: '上报时间' }
            ]}
            onSearch={handleSearch}
            position="right"
            width={300}
            defaultField={isAdminMode ? 'user_id' : 'account_id'} // 直接根据当前模式设置
          />
        </Box>

        {/* 数据表格 */}
        <Box sx={{ mb: 3 }}>
          <DataTable
            columns={getColumns()}
            data={filteredInstances} // 使用过滤后的数据
            selectable
            rowKey="instance_id"
            onSelectionChange={handleSelectionChange}
            searchable={false} // 关闭表格内部搜索功能
          />
        </Box>

        {/* 自动刷新状态提示 */}
        <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
          {autoRefreshEnabled 
            ? '数据每5秒自动刷新一次，最多持续5分钟' 
            : '自动刷新已停止，点击刷新按钮可重新启动'}
        </Typography>
        
        {/* 管理员模式提示 */}
        {isAdmin && isAdminMode && (
          <Typography variant="body2" color="primary" sx={{ mt: 1 }}>
            当前处于管理员模式，显示所有用户的实例
          </Typography>
        )}
      </DashboardContent>
    </>
  );
}