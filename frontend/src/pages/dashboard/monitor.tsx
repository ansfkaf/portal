// src/pages/dashboard/monitor.tsx
import { TableSearch } from 'src/components/table/table-search';
import { useEffect, useState, useMemo } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import TextField from '@mui/material/TextField';
import Switch from '@mui/material/Switch';
import FormControlLabel from '@mui/material/FormControlLabel';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Tooltip from '@mui/material/Tooltip';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import axiosInstance from 'src/lib/axios';
import { useIsAdmin } from 'src/auth/hooks/use-is-admin';
import { DataTable, type Column } from 'src/components/table/data-table';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
// 导入新的管理员模式组件和钩子
import { AdminModeToggle, useAdminMode } from 'src/components/admin-mode-toggle';

// 监控设置数据接口
interface MonitorConfig {
  id: string;
  user_id: string;
  threshold: number;
  jp_threshold: number; // 新增日本区阈值
  sg_threshold: number; // 新增新加坡区阈值
  is_enabled: boolean;
  is_tg_enabled: boolean;
  tg_user_id: string | null;
  is_ip_range_enabled: boolean;
  ip_range: string;
  jp_ip_range: string; // 新增日本区IP范围
  sg_ip_range: string; // 新增新加坡区IP范围
}

// 扩展类型以适应DataTable组件
interface MonitorConfigDisplay extends MonitorConfig {
  user_id_display: string;
}

// 管理员模式下的列表响应接口
interface MonitorListResponse {
  list: MonitorConfig[];
  total: number;
}

// TG绑定响应接口
interface TgBindResponse {
  binding_code: string;
  binding_url: string;
  bot_username: string;
}

// 主动检测响应接口
interface DetectResponse {
  message: string;
  results: Array<{
    UserID: string;
    Count: number;
  }>;
}

// API响应接口
interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

export default function MonitorPage() {
  // 从本地存储获取用户信息
  const [user, setUser] = useState<{ id: string; email: string; isAdmin: number; role: string } | null>(null);
  const isAdmin = useIsAdmin(); // 获取当前用户是否为管理员
  
  // 使用自定义钩子管理管理员模式状态
  const { isAdminMode } = useAdminMode();
  
  // 所有用户的监控设置列表（仅在管理员模式下使用）
  const [monitorConfigList, setMonitorConfigList] = useState<MonitorConfigDisplay[]>([]);
  
  // 编辑对话框状态
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingConfig, setEditingConfig] = useState<MonitorConfig | null>(null);
  const [thresholdEditError, setThresholdEditError] = useState('');
  const [jpThresholdEditError, setJpThresholdEditError] = useState(''); // 新增日本区阈值编辑错误
  const [sgThresholdEditError, setSgThresholdEditError] = useState(''); // 新增新加坡区阈值编辑错误
  const [tgUserIdEditError, setTgUserIdEditError] = useState('');
  const [ipRangeEditError, setIpRangeEditError] = useState('');
  const [jpIpRangeEditError, setJpIpRangeEditError] = useState(''); // 新增日本区IP范围编辑错误
  const [sgIpRangeEditError, setSgIpRangeEditError] = useState(''); // 新增新加坡区IP范围编辑错误
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchField, setSearchField] = useState('user_id_display'); // 默认按用户ID搜索
  
  // 组件加载时从本地存储获取用户数据
  useEffect(() => {
    try {
      const userDataStr = localStorage.getItem('user_data');
      if (userDataStr) {
        const userData = JSON.parse(userDataStr);
        setUser(userData);
      }
    } catch (error) {
      console.error('获取用户数据失败:', error);
    }
  }, []);
  
  // 监控设置状态
  const [monitorConfig, setMonitorConfig] = useState<MonitorConfig>({
    id: '',
    user_id: '',
    threshold: 1,
    jp_threshold: 1, // 新增日本区阈值默认值
    sg_threshold: 1, // 新增新加坡区阈值默认值
    is_enabled: false,
    is_tg_enabled: false,
    tg_user_id: null,
    is_ip_range_enabled: false,
    ip_range: '',
    jp_ip_range: '', // 新增日本区IP范围默认值
    sg_ip_range: '' // 新增新加坡区IP范围默认值
  });
  
  // TG绑定码状态
  const [tgBindData, setTgBindData] = useState<TgBindResponse | null>(null);
  
  // 验证错误状态
  const [thresholdError, setThresholdError] = useState('');
  const [jpThresholdError, setJpThresholdError] = useState(''); // 新增日本区阈值错误状态
  const [sgThresholdError, setSgThresholdError] = useState(''); // 新增新加坡区阈值错误状态
  const [ipRangeError, setIpRangeError] = useState('');
  const [jpIpRangeError, setJpIpRangeError] = useState(''); // 新增日本区IP范围错误状态
  const [sgIpRangeError, setSgIpRangeError] = useState(''); // 新增新加坡区IP范围错误状态
  
  // 加载状态
  const [loading, setLoading] = useState(false);
  
  // 使用提示组件
  const { addAlert } = useAlert();

  // 创建用于表格展示的列定义
  const columns: Column[] = [
    {
      id: 'user_id_display',
      label: '用户ID',
      width: '80px',
      sortable: true,
      format: (value) => value,
      sortType: 'numeric-string'
    },
    {
      id: 'is_enabled',
      label: '监控状态',
      sortable: true,
      format: (value) => value ? '已启用' : '已禁用'
    },
    {
      id: 'threshold',
      label: '香港',
      sortable: true
    },
    {
      id: 'jp_threshold',
      label: '日本',
      sortable: true
    },
    {
      id: 'sg_threshold',
      label: '新加坡',
      sortable: true
    },
    {
      id: 'is_tg_enabled',
      label: 'TG通知',
      sortable: true,
      format: (value) => value ? '已启用' : '已禁用'
    },
    {
      id: 'tg_user_id',
      label: 'TG用户ID',
      sortable: true,
      format: (value) => value || '未绑定'
    },
    {
      id: 'is_ip_range_enabled',
      label: 'IP限制',
      sortable: true,
      format: (value) => value ? '已启用' : '已禁用'
    },
    {
      id: 'ip_range',
      label: 'IP段(香港)',
      sortable: true,
      format: (value) => value || '未设置'
    },
    {
      id: 'jp_ip_range',
      label: 'IP段(日本)',
      sortable: true,
      format: (value) => value || '未设置'
    },
    {
      id: 'sg_ip_range',
      label: 'IP段(新加坡)',
      sortable: true,
      format: (value) => value || '未设置'
    },
    {
      id: 'user_id',
      label: '操作',
      width: '100px',
      format: (value) => (
        <Button 
          variant="contained" 
          color="primary" 
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            handleOpenEditDialog(value as string);
          }}
        >
          编辑
        </Button>
      )
    }
  ];

  // 搜索框设置 - 筛选可搜索的列
  const searchColumns = useMemo(() => {
    return columns.filter(col => 
      col.id !== 'user_id' && // 排除操作列
      (col.id === 'user_id_display' || // 包含用户ID显示
       col.id === 'threshold' || // 包含香港阈值
       col.id === 'jp_threshold' || // 包含日本阈值
       col.id === 'sg_threshold' || // 包含新加坡阈值
       col.id === 'is_enabled' || // 包含监控状态
       col.id === 'is_tg_enabled' || // 包含TG通知
       col.id === 'tg_user_id' || // 包含TG用户ID
       col.id === 'is_ip_range_enabled' || // 包含IP限制
       col.id === 'ip_range' || // 包含香港IP段
       col.id === 'jp_ip_range' || // 包含日本IP段
       col.id === 'sg_ip_range') // 包含新加坡IP段
    );
  }, []);

  // 获取所有用户的监控设置（管理员模式）
  const fetchAllMonitorConfigs = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.post<ApiResponse<MonitorListResponse>>('/monitor/admin');
      
      if (response.data.code === 200 && response.data.data.list) {
        // 处理数据，添加用于显示的字段
        const processedList = response.data.data.list.map(item => ({
          ...item,
          user_id_display: item.user_id // 复制user_id到user_id_display字段用于显示
        }));
        setMonitorConfigList(processedList);
      } else {
        addAlert('error', '获取所有用户监控设置失败');
      }
    } catch (error) {
      addAlert('error', '获取监控设置列表请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取当前用户的监控设置（普通模式）
  const fetchMonitorConfig = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.get<ApiResponse<{config: MonitorConfig}>>('/monitor');
      
      if (response.data.code === 200 && response.data.data.config) {
        setMonitorConfig(response.data.data.config);
      } else {
        addAlert('error', '获取监控设置失败');
      }
    } catch (error) {
      addAlert('error', '获取监控设置请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 组件加载时获取数据
  useEffect(() => {
    // 根据当前模式获取相应的数据
    if (isAdmin && isAdminMode) {
      fetchAllMonitorConfigs();
    } else {
      fetchMonitorConfig();
    }
  }, [isAdmin, isAdminMode]);

  // 处理监控开关状态变化
  const handleSwitchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setMonitorConfig(prev => ({
      ...prev,
      is_enabled: event.target.checked
    }));
  };

  // 处理TG通知开关状态变化
  const handleTgSwitchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setMonitorConfig(prev => ({
      ...prev,
      is_tg_enabled: event.target.checked
    }));
  };

  // 处理IP范围开关状态变化
  const handleIpRangeEnabledChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setMonitorConfig(prev => ({
      ...prev,
      is_ip_range_enabled: event.target.checked
    }));
  };

  // 处理香港IP范围输入变化
  const handleIpRangeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // 清除之前的错误
    setIpRangeError('');
    
    // 验证IP范围格式（简单验证，可以根据需要调整）
    if (value && !/^[0-9.]+$/.test(value)) {
      setIpRangeError('IP范围格式不正确，只能包含数字和点');
      return;
    }
    
    setMonitorConfig(prev => ({
      ...prev,
      ip_range: value
    }));
  };
  
  // 处理日本IP范围输入变化
  const handleJpIpRangeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // 清除之前的错误
    setJpIpRangeError('');
    
    // 验证IP范围格式
    if (value && !/^[0-9.]+$/.test(value)) {
      setJpIpRangeError('IP范围格式不正确，只能包含数字和点');
      return;
    }
    
    setMonitorConfig(prev => ({
      ...prev,
      jp_ip_range: value
    }));
  };
  
  // 处理新加坡IP范围输入变化
  const handleSgIpRangeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // 清除之前的错误
    setSgIpRangeError('');
    
    // 验证IP范围格式
    if (value && !/^[0-9.]+$/.test(value)) {
      setSgIpRangeError('IP范围格式不正确，只能包含数字和点');
      return;
    }
    
    setMonitorConfig(prev => ({
      ...prev,
      sg_ip_range: value
    }));
  };

  // 处理香港阈值变化
  const handleThresholdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // 清除之前的错误
    setThresholdError('');
    
    // 验证输入
    if (value === '') {
      setMonitorConfig(prev => ({
        ...prev,
        threshold: 0
      }));
      return;
    }
    
    const numValue = parseInt(value, 10);
    
    // 验证是否为整数
    if (isNaN(numValue) || numValue.toString() !== value) {
      setThresholdError('请输入整数值');
      return;
    }
    
    // 验证是否大于等于0
    if (numValue < 0) {
      setThresholdError('阈值不能小于0');
      return;
    }
    
    setMonitorConfig(prev => ({
      ...prev,
      threshold: numValue
    }));
  };
  
  // 处理日本阈值变化
  const handleJpThresholdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // 清除之前的错误
    setJpThresholdError('');
    
    // 验证输入
    if (value === '') {
      setMonitorConfig(prev => ({
        ...prev,
        jp_threshold: 0
      }));
      return;
    }
    
    const numValue = parseInt(value, 10);
    
    // 验证是否为整数
    if (isNaN(numValue) || numValue.toString() !== value) {
      setJpThresholdError('请输入整数值');
      return;
    }
    
    // 验证是否大于等于0
    if (numValue < 0) {
      setJpThresholdError('阈值不能小于0');
      return;
    }
    
    setMonitorConfig(prev => ({
      ...prev,
      jp_threshold: numValue
    }));
  };
  
  // 处理新加坡阈值变化
  const handleSgThresholdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // 清除之前的错误
    setSgThresholdError('');
    
    // 验证输入
    if (value === '') {
      setMonitorConfig(prev => ({
        ...prev,
        sg_threshold: 0
      }));
      return;
    }
    
    const numValue = parseInt(value, 10);
    
    // 验证是否为整数
    if (isNaN(numValue) || numValue.toString() !== value) {
      setSgThresholdError('请输入整数值');
      return;
    }
    
    // 验证是否大于等于0
    if (numValue < 0) {
      setSgThresholdError('阈值不能小于0');
      return;
    }
    
    setMonitorConfig(prev => ({
      ...prev,
      sg_threshold: numValue
    }));
  };
// 获取TG绑定码
const handleGetBindCode = async () => {
  try {
    setLoading(true);
    const response = await axiosInstance.get<ApiResponse<TgBindResponse>>('/monitor/tg/bind');
    
    if (response.data.code === 200 && response.data.data.binding_code) {
      setTgBindData({
        binding_code: response.data.data.binding_code,
        binding_url: response.data.data.binding_url,
        bot_username: response.data.data.bot_username
      });
    } else {
      addAlert('error', '获取TG绑定码失败');
    }
  } catch (error) {
    addAlert('error', '获取TG绑定码请求失败');
  } finally {
    setLoading(false);
  }
};

// 解绑TG账号
const handleUnbindTg = async () => {
  try {
    setLoading(true);
    const response = await axiosInstance.post<ApiResponse>('/monitor/tg/unbind');
    
    if (response.data.code === 200) {
      addAlert('success', response.data.data.message || 'TG账号解绑成功');
      // 刷新配置数据
      fetchMonitorConfig();
    } else {
      addAlert('error', '解绑TG账号失败');
    }
  } catch (error) {
    addAlert('error', '解绑TG账号请求失败');
  } finally {
    setLoading(false);
  }
};

// 触发IP段检测（普通用户）
const handleCheckIpRange = async () => {
  try {
    setLoading(true);
    const response = await axiosInstance.post<ApiResponse>('/monitor/check-ip');
    
    if (response.data.code === 200) {
      addAlert('success', response.data.data.message || '已触发IP范围检查，请稍后查看结果');
    } else {
      addAlert('error', '触发IP段检测失败');
    }
  } catch (error) {
    addAlert('error', '触发IP段检测请求失败');
  } finally {
    setLoading(false);
  }
};

// 触发IP段检测（管理员）
const handleAdminCheckIpRange = async () => {
  try {
    setLoading(true);
    const response = await axiosInstance.post<ApiResponse>('/monitor/admin/check-ip');
    
    if (response.data.code === 200) {
      addAlert('success', response.data.data.message || '已触发IP范围检查，请稍后查看结果');
    } else {
      addAlert('error', '触发IP段检测失败');
    }
  } catch (error) {
    addAlert('error', '触发IP段检测请求失败');
  } finally {
    setLoading(false);
  }
};

// 保存设置（普通模式）
const handleSave = async () => {
  // 验证输入
  if (thresholdError || ipRangeError || jpThresholdError || sgThresholdError || jpIpRangeError || sgIpRangeError) {
    addAlert('warning', '请修正输入错误后再保存');
    return;
  }
  
  try {
    setLoading(true);
    
    const response = await axiosInstance.post<ApiResponse>('/monitor', {
      threshold: monitorConfig.threshold,
      jp_threshold: monitorConfig.jp_threshold,
      sg_threshold: monitorConfig.sg_threshold,
      is_enabled: monitorConfig.is_enabled,
      is_tg_enabled: monitorConfig.is_tg_enabled,
      is_ip_range_enabled: monitorConfig.is_ip_range_enabled,
      ip_range: monitorConfig.ip_range,
      jp_ip_range: monitorConfig.jp_ip_range,
      sg_ip_range: monitorConfig.sg_ip_range
    });
    
    if (response.data.code === 200) {
      addAlert('success', response.data.data.message || '监控设置保存成功');
    } else {
      addAlert('error', '保存监控设置失败');
    }
  } catch (error) {
    addAlert('error', '保存监控设置请求失败');
  } finally {
    setLoading(false);
  }
};

// 立即触发主动检测
const handleTriggerDetection = async () => {
  try {
    setLoading(true);
    const response = await axiosInstance.post<ApiResponse<DetectResponse>>('/monitor/admin/detect');
    
    if (response.data.code === 200) {
      // 获取检测结果
      const { message, results } = response.data.data;
      
      if (results.length > 0) {
        // 如果有需要补机的用户，显示详细信息
        addAlert('success', `${message}: ${results.length}个用户需要补机`);
      } else {
        // 如果没有需要补机的用户
        addAlert('success', message);
      }
      
      // 刷新配置和监控状态
      fetchAllMonitorConfigs();
    } else {
      addAlert('error', '触发检测失败');
    }
  } catch (error) {
    addAlert('error', '触发检测请求失败');
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
      addAlert('success', response.data.data.message || '已清空所有补机历史记录并重置账号冷却状态');
      // 刷新配置数据
      fetchAllMonitorConfigs();
    } else {
      addAlert('error', '清空补机历史失败');
    }
  } catch (error) {
    addAlert('error', '清空补机历史请求失败');
  } finally {
    setLoading(false);
  }
};

// 备份监控设置并关闭所有通知
const handleBackupSettings = async () => {
  try {
    setLoading(true);
    const response = await axiosInstance.post<ApiResponse>('/monitor/admin/backup');
    
    if (response.data.code === 200) {
      addAlert('success', response.data.data.message || '已备份所有监控配置并临时关闭所有TG通知');
      // 刷新配置数据
      fetchAllMonitorConfigs();
    } else {
      addAlert('error', '备份监控设置失败');
    }
  } catch (error) {
    addAlert('error', '备份监控设置请求失败');
  } finally {
    setLoading(false);
  }
};

// 恢复监控设置
const handleRestoreSettings = async () => {
  try {
    setLoading(true);
    const response = await axiosInstance.post<ApiResponse>('/monitor/admin/restore');
    
    if (response.data.code === 200) {
      addAlert('success', response.data.data.message || '已成功恢复所有监控配置');
      // 刷新配置数据
      fetchAllMonitorConfigs();
    } else {
      addAlert('error', '恢复监控设置失败');
    }
  } catch (error) {
    addAlert('error', '恢复监控设置请求失败');
  } finally {
    setLoading(false);
  }
};

// 关闭编辑对话框
const handleCloseEditDialog = () => {
  setEditDialogOpen(false);
  setEditingConfig(null);
};

// 处理编辑对话框中的香港阈值变化
const handleEditThresholdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  const value = event.target.value;
  
  // 清除之前的错误
  setThresholdEditError('');
  
  if (!editingConfig) return;
  
  // 验证输入
  if (value === '') {
    setEditingConfig({
      ...editingConfig,
      threshold: 0
    });
    return;
  }
  
  const numValue = parseInt(value, 10);
  
  // 验证是否为整数
  if (isNaN(numValue) || numValue.toString() !== value) {
    setThresholdEditError('请输入整数值');
    return;
  }
  
  // 验证是否大于等于0
  if (numValue < 0) {
    setThresholdEditError('阈值不能小于0');
    return;
  }
  
  setEditingConfig({
    ...editingConfig,
    threshold: numValue
  });
};

// 处理编辑对话框中的日本阈值变化
const handleEditJpThresholdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  const value = event.target.value;
  
  // 清除之前的错误
  setJpThresholdEditError('');
  
  if (!editingConfig) return;
  
  // 验证输入
  if (value === '') {
    setEditingConfig({
      ...editingConfig,
      jp_threshold: 0
    });
    return;
  }
  
  const numValue = parseInt(value, 10);
  
  // 验证是否为整数
  if (isNaN(numValue) || numValue.toString() !== value) {
    setJpThresholdEditError('请输入整数值');
    return;
  }
  
  // 验证是否大于等于0
  if (numValue < 0) {
    setJpThresholdEditError('阈值不能小于0');
    return;
  }
  
  setEditingConfig({
    ...editingConfig,
    jp_threshold: numValue
  });
};

// 处理编辑对话框中的新加坡阈值变化
const handleEditSgThresholdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  const value = event.target.value;
  
  // 清除之前的错误
  setSgThresholdEditError('');
  
  if (!editingConfig) return;
  
  // 验证输入
  if (value === '') {
    setEditingConfig({
      ...editingConfig,
      sg_threshold: 0
    });
    return;
  }
  
  const numValue = parseInt(value, 10);
  
  // 验证是否为整数
  if (isNaN(numValue) || numValue.toString() !== value) {
    setSgThresholdEditError('请输入整数值');
    return;
  }
  
  // 验证是否大于等于0
  if (numValue < 0) {
    setSgThresholdEditError('阈值不能小于0');
    return;
  }
  
  setEditingConfig({
    ...editingConfig,
    sg_threshold: numValue
  });
};

// 处理编辑对话框中的TG账号ID变化
const handleEditTgUserIdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  const value = event.target.value;
  
  // 清除之前的错误
  setTgUserIdEditError('');
  
  if (!editingConfig) return;
  
  // TG ID只能包含数字
  if (value !== '' && !/^\d*$/.test(value)) {
    setTgUserIdEditError('TG用户ID只能包含数字');
    return;
  }
  
  setEditingConfig({
    ...editingConfig,
    tg_user_id: value
  });
};

// 处理编辑对话框中的香港IP范围变化
const handleEditIpRangeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  const value = event.target.value;
  
  // 清除之前的错误
  setIpRangeEditError('');
  
  if (!editingConfig) return;
  
  // 验证IP范围格式
  if (value && !/^[0-9.]+$/.test(value)) {
    setIpRangeEditError('IP范围格式不正确，只能包含数字和点');
    return;
  }
  
  setEditingConfig({
    ...editingConfig,
    ip_range: value
  });
};

// 处理编辑对话框中的日本IP范围变化
const handleEditJpIpRangeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  const value = event.target.value;
  
  // 清除之前的错误
  setJpIpRangeEditError('');
  
  if (!editingConfig) return;
  
  // 验证IP范围格式
  if (value && !/^[0-9.]+$/.test(value)) {
    setJpIpRangeEditError('IP范围格式不正确，只能包含数字和点');
    return;
  }
  
  setEditingConfig({
    ...editingConfig,
    jp_ip_range: value
  });
};

// 处理编辑对话框中的新加坡IP范围变化
const handleEditSgIpRangeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  const value = event.target.value;
  
  // 清除之前的错误
  setSgIpRangeEditError('');
  
  if (!editingConfig) return;
  
  // 验证IP范围格式
  if (value && !/^[0-9.]+$/.test(value)) {
    setSgIpRangeEditError('IP范围格式不正确，只能包含数字和点');
    return;
  }
  
  setEditingConfig({
    ...editingConfig,
    sg_ip_range: value
  });
};

// 处理编辑对话框中的监控开关变化
const handleEditEnabledChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  if (!editingConfig) return;
  
  setEditingConfig({
    ...editingConfig,
    is_enabled: event.target.checked
  });
};

// 处理编辑对话框中的TG通知开关变化
const handleEditTgEnabledChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  if (!editingConfig) return;
  
  setEditingConfig({
    ...editingConfig,
    is_tg_enabled: event.target.checked
  });
};

// 处理编辑对话框中的IP范围开关变化
const handleSaveEdit = async () => {
  if (!editingConfig) return;
  
  // 验证输入
  if (thresholdEditError || tgUserIdEditError || ipRangeEditError || 
      jpThresholdEditError || sgThresholdEditError || 
      jpIpRangeEditError || sgIpRangeEditError) {
    addAlert('warning', '请修正输入错误后再保存');
    return;
  }
  
  try {
    setLoading(true);
    
    // 创建请求数据对象，确保包含所有必要字段
    const updateData = {
      user_id: editingConfig.user_id,
      threshold: editingConfig.threshold,
      jp_threshold: editingConfig.jp_threshold,
      sg_threshold: editingConfig.sg_threshold,
      is_enabled: editingConfig.is_enabled,
      is_tg_enabled: editingConfig.is_tg_enabled,
      tg_user_id: editingConfig.tg_user_id || '',
      is_ip_range_enabled: editingConfig.is_ip_range_enabled,
      ip_range: editingConfig.ip_range,
      jp_ip_range: editingConfig.jp_ip_range,
      sg_ip_range: editingConfig.sg_ip_range
    };
    
    console.log('发送更新请求数据:', updateData); // 调试日志
    
    const response = await axiosInstance.post<ApiResponse>('/monitor/admin/update', updateData);
    
    if (response.data.code === 200) {
      addAlert('success', response.data.data.message || '监控设置更新成功');
      // 关闭对话框
      handleCloseEditDialog();
      // 刷新列表数据
      fetchAllMonitorConfigs();
    } else {
      addAlert('error', `更新监控设置失败: ${response.data.message}`);
    }
  } catch (error) {
    console.error('更新设置错误:', error);
    addAlert('error', '更新监控设置请求失败');
  } finally {
    setLoading(false);
  }
};
// 处理搜索
const handleSearch = (keyword: string, field: string) => {
  setSearchKeyword(keyword);
  setSearchField(field);
};

// 根据搜索条件过滤监控配置
const filteredMonitorConfigList = useMemo(() => {
  if (!searchKeyword || !searchField) return monitorConfigList;
  
  return monitorConfigList.filter(config => {
    const value = config[searchField as keyof MonitorConfigDisplay];
    if (value === null || value === undefined) return false;
    
    return String(value).toLowerCase().includes(searchKeyword.toLowerCase());
  });
}, [monitorConfigList, searchKeyword, searchField]);

// 添加缺失的函数：处理编辑对话框中的IP范围开关变化
const handleEditIpRangeEnabledChange = (event: React.ChangeEvent<HTMLInputElement>) => {
  if (!editingConfig) return;
  
  setEditingConfig({
    ...editingConfig,
    is_ip_range_enabled: event.target.checked
  });
};

// 打开编辑对话框
const handleOpenEditDialog = (userId: string) => {
  // 从缓存的列表中查找对应用户的配置
  const config = monitorConfigList.find(cfg => cfg.user_id === userId);
  
  if (config) {
    console.log('编辑用户配置:', config); // 调试日志
    setEditingConfig({...config});
    
    // 重置所有错误状态
    setThresholdEditError('');
    setJpThresholdEditError('');
    setSgThresholdEditError('');
    setTgUserIdEditError('');
    setIpRangeEditError('');
    setJpIpRangeEditError('');
    setSgIpRangeEditError('');
    setEditDialogOpen(true);
  } else {
    addAlert('error', '找不到用户配置数据');
  }
};

return (
  <>
    <Helmet>
      <title>{`监控管理 | ${CONFIG.appName}`}</title>
    </Helmet>

    <DashboardContent maxWidth="xl">
      {/* 标题部分和管理员模式切换 */}
      <Stack 
        direction="row" 
        alignItems="center" 
        justifyContent="space-between" 
        sx={{ mb: { xs: 3, md: 5 } }}
      >
        <Typography variant="h4">
          监控管理
        </Typography>
        
        {/* 使用AdminModeToggle组件 */}
        <AdminModeToggle 
          visible={isAdmin}
        />
      </Stack>

      {/* 管理员模式内容 */}
      {isAdmin && isAdminMode ? (
        <>
          {/* 管理员操作按钮 - 包含新增的部署相关按钮 */}
          {/* 按钮组和搜索框在同一行 */}
          <Box sx={{ 
            display: 'flex', 
            flexDirection: { xs: 'column', md: 'row' }, 
            justifyContent: 'space-between',
            alignItems: { xs: 'stretch', md: 'center' },
            mb: 3 
          }}>
            {/* 管理员操作按钮 */}
            <Stack 
              direction="row" 
              spacing={2} 
              sx={{ 
                mb: { xs: 2, md: 0 },
                flexWrap: 'wrap',
                gap: 1
              }}
            >
              <CustomButton 
                onClick={handleTriggerDetection}
                disabled={loading}
              >
                立即检测
              </CustomButton>
              <CustomButton 
                onClick={handleClearHistory}
                disabled={loading}
              >
                清理队列
              </CustomButton>
              <CustomButton 
                onClick={handleBackupSettings}
                disabled={loading}
                color="warning"
              >
                关闭通知
              </CustomButton>
              <CustomButton 
                onClick={handleRestoreSettings}
                disabled={loading}
                color="info"
              >
                恢复通知
              </CustomButton>
              <CustomButton 
                onClick={handleAdminCheckIpRange}
                disabled={loading}
                color="success"
              >
                检测IP段
              </CustomButton>
            </Stack>
            
            {/* 搜索框 */}
            <TableSearch
              columns={[
                { id: 'user_id_display', label: '用户ID' },
                { id: 'is_enabled', label: '监控状态' },
                { id: 'threshold', label: '阈值(香港)' }, 
                { id: 'jp_threshold', label: '阈值(日本)' },
                { id: 'sg_threshold', label: '阈值(新加坡)' },
                { id: 'is_tg_enabled', label: 'TG通知' },
                { id: 'tg_user_id', label: 'TG用户ID' },
                { id: 'is_ip_range_enabled', label: 'IP限制' },
                { id: 'ip_range', label: 'IP段(香港)' },
                { id: 'jp_ip_range', label: 'IP段(日本)' },
                { id: 'sg_ip_range', label: 'IP段(新加坡)' }
              ]}
              onSearch={handleSearch}
              position="right"
              width={300}
              defaultField="user_id_display"
            />
          </Box>
          
          {/* 数据表格 */}
          <Box sx={{ width: '100%', mb: 2 }}>
            <Card>
              <CardContent>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  点击表格行可编辑对应用户的监控设置
                </Typography>
                <DataTable
                  columns={columns}
                  data={filteredMonitorConfigList} // 使用过滤后的数据
                  rowKey="id"
                  searchable={false} // 关闭表格内部搜索功能
                />
              </CardContent>
            </Card>
          </Box>
          
          {/* 编辑对话框 */}
          <Dialog 
            open={editDialogOpen} 
            onClose={handleCloseEditDialog}
            maxWidth="sm"
            fullWidth
          >
            <DialogTitle>
              编辑用户监控设置 (用户ID: {editingConfig?.user_id})
            </DialogTitle>
            <DialogContent>
              <Stack spacing={3} sx={{ mt: 1 }}>
                {/* 监控开关 */}
                <FormControlLabel
                  control={
                    <Switch
                      checked={editingConfig?.is_enabled || false}
                      onChange={handleEditEnabledChange}
                      color="primary"
                    />
                  }
                  label="启用监控主程序"
                />

<Typography variant="subtitle1" sx={{ fontWeight: 'bold', mt: 1 }}>
  各地区阈值设置
</Typography>
<Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2 }}>
  {/* 香港阈值设置 */}
  <TextField
    label="香港区阈值"
    value={editingConfig?.threshold || 0}
    onChange={handleEditThresholdChange}
    error={!!thresholdEditError}
    helperText={thresholdEditError || "低于此阈值将触发补机"}
    sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
    type="number"
    InputProps={{ inputProps: { min: 0 } }}
    size="small"
  />

  {/* 日本阈值设置 */}
  <TextField
    label="日本区阈值"
    value={editingConfig?.jp_threshold || 0}
    onChange={handleEditJpThresholdChange}
    error={!!jpThresholdEditError}
    helperText={jpThresholdEditError || "低于此阈值将触发补机"}
    sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
    type="number"
    InputProps={{ inputProps: { min: 0 } }}
    size="small"
  />

  {/* 新加坡阈值设置 */}
  <TextField
    label="新加坡区阈值"
    value={editingConfig?.sg_threshold || 0}
    onChange={handleEditSgThresholdChange}
    error={!!sgThresholdEditError}
    helperText={sgThresholdEditError || "低于此阈值将触发补机"}
    sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
    type="number"
    InputProps={{ inputProps: { min: 0 } }}
    size="small"
  />
</Box>

  <Divider sx={{ my: 0.5 }} />

  {/* TG通知设置 */}
  <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>
    Telegram 通知设置
  </Typography>

  {/* TG通知开关 */}
  <FormControlLabel
    control={
      <Switch
        checked={editingConfig?.is_tg_enabled || false}
        onChange={handleEditTgEnabledChange}
        color="primary"
      />
    }
    label="启用Telegram通知"
  />

  {/* TG用户ID设置 */}
  <TextField
    label="Telegram用户ID"
    value={editingConfig?.tg_user_id || ''}
    onChange={handleEditTgUserIdChange}
    error={!!tgUserIdEditError}
    helperText={tgUserIdEditError || "设置Telegram用户ID，留空表示未绑定"}
    fullWidth
  />

  <Divider sx={{ my: 1 }} />

  {/* IP范围设置 */}
  <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>
    IP范围限制设置
  </Typography>

  {/* IP范围开关 */}
  <FormControlLabel
    control={
      <Switch
        checked={editingConfig?.is_ip_range_enabled || false}
        onChange={handleEditIpRangeEnabledChange}
        color="primary"
      />
    }
    label="启用IP范围限制"
  />

  <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2 }}>
    {/* 香港IP范围输入 */}
    <TextField
      label="香港区IP范围"
      value={editingConfig?.ip_range || ''}
      onChange={handleEditIpRangeChange}
      error={!!ipRangeEditError}
      helperText={ipRangeEditError || "例如: 43 或 43.19"}
      sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
      size="small"
    />

    {/* 日本IP范围输入 */}
    <TextField
      label="日本区IP范围"
      value={editingConfig?.jp_ip_range || ''}
      onChange={handleEditJpIpRangeChange}
      error={!!jpIpRangeEditError}
      helperText={jpIpRangeEditError || "例如: 35 或 35.72"}
      sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
      size="small"
    />

    {/* 新加坡IP范围输入 */}
    <TextField
      label="新加坡区IP范围"
      value={editingConfig?.sg_ip_range || ''}
      onChange={handleEditSgIpRangeChange}
      error={!!sgIpRangeEditError}
      helperText={sgIpRangeEditError || "例如: 13 或 13.228"}
      sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
      size="small"
    />
  </Box>
</Stack>
</DialogContent>
<DialogActions>
<Button onClick={handleCloseEditDialog}>取消</Button>
<Button 
  variant="contained" 
  onClick={handleSaveEdit}
  disabled={loading || !!thresholdEditError || !!jpThresholdEditError || 
            !!sgThresholdEditError || !!tgUserIdEditError || 
            !!ipRangeEditError || !!jpIpRangeEditError || !!sgIpRangeEditError}
>
  保存更改
</Button>
</DialogActions>
</Dialog>

{/* 管理员模式提示 */}
<Box sx={{ mt: 1, p: 2, bgcolor: 'background.neutral', borderRadius: 1 }}>
<Typography variant="body2" color="primary">
当前处于管理员模式，可管理所有用户的监控设置
</Typography>
<Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
• 备份并关闭通知：在系统维护或部署前使用，将保存当前所有监控设置并关闭所有TG通知<br />
• 恢复通知设置：在系统维护或部署完成后使用，将恢复之前备份的监控设置<br />
• 检测IP段：触发系统对所有用户设置的IP范围进行检测，确认IP段是否有效
</Typography>
</Box>
</>
) : (
/* 普通模式内容 */
<>
<Box sx={{ width: '100%', mb: 2 }}>
<Card>
<CardContent>
  <Stack spacing={2} sx={{ maxWidth: 600 }}>
    {/* 监控开关 */}
    <FormControlLabel
      control={
        <Switch
          checked={monitorConfig.is_enabled}
          onChange={handleSwitchChange}
          color="primary"
        />
      }
      label="启用监控主程序"
      sx={{ fontSize: "1rem" }}
    />

    <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>
      各地区阈值设置
    </Typography>

    <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2 }}>
  {/* 香港阈值 */}
  <TextField
    label="香港区阈值"
    value={monitorConfig.threshold} // 修改这里：使用monitorConfig而不是editingConfig
    onChange={handleThresholdChange}
    error={!!thresholdError}
    helperText={thresholdError || "低于此阈值将触发补机"}
    type="number"
    InputProps={{
      inputProps: { min: 0 }
    }}
    size="small"
    sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
  />

  {/* 日本阈值 */}
  <TextField
    label="日本区阈值"
    value={monitorConfig.jp_threshold} // 修改这里：使用monitorConfig而不是editingConfig
    onChange={handleJpThresholdChange}
    error={!!jpThresholdError}
    helperText={jpThresholdError || "低于此阈值将触发补机"}
    type="number"
    InputProps={{
      inputProps: { min: 0 }
    }}
    size="small"
    sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
  />

  {/* 新加坡阈值 */}
  <TextField
    label="新加坡区阈值"
    value={monitorConfig.sg_threshold} // 修改这里：使用monitorConfig而不是editingConfig
    onChange={handleSgThresholdChange}
    error={!!sgThresholdError}
    helperText={sgThresholdError || "低于此阈值将触发补机"}
    type="number"
    InputProps={{
      inputProps: { min: 0 }
    }}
    size="small"
    sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
  />
</Box>

        <Divider sx={{ my: 0.5 }} />

        {/* IP范围限制设置 */}
        <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mt: 1 }}>
          IP范围限制设置
        </Typography>

        {/* IP范围开关 */}
        <FormControlLabel
          control={
            <Switch
              checked={monitorConfig.is_ip_range_enabled}
              onChange={handleIpRangeEnabledChange}
              color="primary"
            />
          }
          label="启用IP范围限制"
          sx={{ fontSize: "1rem" }}
        />

        <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2 }}>
          {/* 香港IP范围输入 */}
          <TextField
            label="香港区IP范围"
            value={monitorConfig.ip_range}
            onChange={handleIpRangeChange}
            error={!!ipRangeError}
            helperText={ipRangeError || "例如: 43 或 43.19"}
            sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
            size="small"
          />

          {/* 日本IP范围输入 */}
          <TextField
            label="日本区IP范围"
            value={monitorConfig.jp_ip_range}
            onChange={handleJpIpRangeChange}
            error={!!jpIpRangeError}
            helperText={jpIpRangeError || "例如: 35 或 35.72"}
            sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
            size="small"
          />

          {/* 新加坡IP范围输入 */}
          <TextField
            label="新加坡区IP范围"
            value={monitorConfig.sg_ip_range}
            onChange={handleSgIpRangeChange}
            error={!!sgIpRangeError}
            helperText={sgIpRangeError || "例如: 13 或 13.228"}
            sx={{ "& .MuiFormHelperText-root": { fontSize: "0.75rem" } }}
            size="small"
          />
        </Box>
        
        <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5, fontSize: "0.875rem" }}>
          先点击保存后，才可触发IP检测。由于api速率限制，每分钟仅可自动更换1次ip。
        </Typography>

        {/* IP段检测按钮 */}
        <Button 
          variant="outlined" 
          color="primary" 
          onClick={handleCheckIpRange}
          disabled={loading}
          sx={{ alignSelf: 'flex-start' }}
        >
          触发IP段检测
        </Button>

        <Divider sx={{ my: 0.5 }} />

        {/* TG通知设置 */}
        <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mt: 1 }}>
          Telegram 通知设置
        </Typography>

        {/* TG通知开关 */}
        <FormControlLabel
          control={
            <Switch
              checked={monitorConfig.is_tg_enabled}
              onChange={handleTgSwitchChange}
              color="primary"
            />
          }
          label="启用Telegram通知"
          sx={{ fontSize: "1rem" }}
        />

        {/* TG绑定状态和操作 */}
        <Box>
          <Typography sx={{ fontWeight: 'medium', mb: 0.5 }}>
            {monitorConfig.tg_user_id 
              ? `已绑定Telegram账号: ${monitorConfig.tg_user_id}` 
              : '未绑定Telegram账号'}
          </Typography>
          
          {monitorConfig.tg_user_id ? (
            <Button 
              variant="outlined" 
              color="error" 
              onClick={handleUnbindTg}
              disabled={loading}
              sx={{ mt: 0.5 }}
            >
              解除绑定
            </Button>
          ) : (
            <>
              <Box sx={{ mt: 0.5, mb: 1 }}>
                {tgBindData && (
                  <Typography variant="body2" color="text.secondary">
                    请添加 Telegram 机器人 
                    <Typography 
                      component="span" 
                      fontWeight="bold"
                      color="primary"
                      sx={{ 
                        cursor: 'pointer',
                        mx: 0.5,
                        '&:hover': { textDecoration: 'underline' }
                      }}
                      onClick={() => {
                        navigator.clipboard.writeText(tgBindData.bot_username);
                        addAlert('success', '机器人ID已复制到剪贴板');
                      }}
                    >
                      {tgBindData.bot_username}
                    </Typography>
                    并发送
                    <Typography 
                      component="span" 
                      fontWeight="bold"
                      color="primary"
                      sx={{ 
                        cursor: 'pointer',
                        mx: 0.5,
                        '&:hover': { textDecoration: 'underline' }
                      }}
                      onClick={() => {
                        navigator.clipboard.writeText(tgBindData.binding_url);
                        addAlert('success', '绑定命令已复制到剪贴板');
                      }}
                    >
                      {tgBindData.binding_url}
                    </Typography>
                    完成绑定。绑定成功后，请刷新页面查看绑定状态。
                  </Typography>
                )}
              </Box>
              <Button 
                variant="outlined" 
                color="primary" 
                onClick={handleGetBindCode}
                disabled={loading}
              >
                获取绑定码
              </Button>
            </>
          )}
        </Box>
      </Stack>
    </CardContent>
  </Card>
</Box>

      {/* 保存按钮 */}
      <Box sx={{ display: 'flex', justifyContent: 'flex-start', mt: 1  }}>
          <CustomButton 
            onClick={handleSave} 
            disabled={loading || !!thresholdError || !!jpThresholdError || 
                      !!sgThresholdError || !!ipRangeError || !!jpIpRangeError || 
                      !!sgIpRangeError}
          >
            保存设置
          </CustomButton>
        </Box>
      </>
    )}
  </DashboardContent>
</>
);
}