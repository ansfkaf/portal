// src/pages/dashboard/setting.tsx
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
import MenuItem from '@mui/material/MenuItem';
import InputAdornment from '@mui/material/InputAdornment';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import axiosInstance from 'src/lib/axios';
import { useIsAdmin } from 'src/auth/hooks/use-is-admin';
import { AdminModeToggle, useAdminMode } from 'src/components/admin-mode-toggle';
import { DataTable, type Column } from 'src/components/table/data-table';
import { TableSearch } from 'src/components/table/table-search';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';

// 设置数据接口
interface SettingData {
  user_id: string;
  region: string;
  instance_type: string;
  disk_size: number;
  password: string;
  script: string;
  jp_script: string;
  sg_script: string;
}

// 管理员列表响应接口
interface SettingListResponse {
  list: SettingData[];
  total: number;
}

// API响应接口
interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

// 区域选项
const regionOptions = [
  { value: '香港', code: 'ap-east-1', label: '香港' },
  { value: '日本', code: 'ap-northeast-3', label: '日本' },
  { value: '新加坡', code: 'ap-southeast-1', label: '新加坡' }
];

// 脚本区域标签接口
interface TabPanelProps {
  children?: React.ReactNode;
  value: number;
  index: number;
}

// 脚本区域选项卡面板组件
function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`script-tabpanel-${index}`}
      aria-labelledby={`script-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box sx={{ pt: 2 }}>
          {children}
        </Box>
      )}
    </div>
  );
}

export default function SettingPage() {
  // 设置数据状态
  const [settings, setSettings] = useState<SettingData>({
    user_id: '',
    region: '香港',
    instance_type: 'c5n.large',
    disk_size: 20,
    password: '',
    script: '',
    jp_script: '',
    sg_script: ''
  });
  
  // 脚本区域标签状态
  const [scriptTabValue, setScriptTabValue] = useState(0);
  
  // 检查当前用户是否为管理员
  const isAdmin = useIsAdmin();
  
  // 使用自定义钩子管理管理员模式状态
  const { isAdminMode } = useAdminMode();
  
  // 所有用户的设置列表（仅在管理员模式下使用）
  const [settingsList, setSettingsList] = useState<SettingData[]>([]);
  
  // 编辑对话框状态
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingSettings, setEditingSettings] = useState<SettingData | null>(null);
  const [editScriptTabValue, setEditScriptTabValue] = useState(0);
  const [passwordError, setPasswordError] = useState('');
  
  // 搜索状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchField, setSearchField] = useState('user_id'); // 默认按用户ID搜索
  
  // 加载状态
  const [loading, setLoading] = useState(false);
  
  // 使用提示组件
  const { addAlert } = useAlert();

  // 处理脚本标签切换
  const handleScriptTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setScriptTabValue(newValue);
  };

  // 处理编辑时脚本标签切换
  const handleEditScriptTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setEditScriptTabValue(newValue);
  };

  // 获取所有用户的设置（管理员模式）
  const fetchAllSettings = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.post<ApiResponse<SettingListResponse>>('/setting/admin');
      
      if (response.data.code === 200 && response.data.data.list) {
        setSettingsList(response.data.data.list);
      } else {
        addAlert('error', '获取所有用户设置失败');
      }
    } catch (error) {
      addAlert('error', '获取设置列表请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取当前用户的设置（普通模式）
  const fetchSettings = async () => {
    try {
      setLoading(true);
      const response = await axiosInstance.get<ApiResponse<SettingData>>('/setting');
      
      if (response.data.code === 200 && typeof response.data.data !== 'string') {
        setSettings(response.data.data as SettingData);
      } else {
        addAlert('error', '获取设置数据失败');
      }
    } catch (error) {
      addAlert('error', '获取设置数据请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 组件加载时获取数据
  useEffect(() => {
    // 根据当前模式获取相应的数据
    if (isAdmin && isAdminMode) {
      fetchAllSettings();
    } else {
      fetchSettings();
    }
  }, [isAdmin, isAdminMode]);

  // 处理输入变化
  const handleChange = (field: keyof SettingData) => (event: React.ChangeEvent<HTMLInputElement>) => {
    setSettings(prev => ({
      ...prev,
      [field]: event.target.value
    }));
  };

  // 处理搜索
  const handleSearch = (keyword: string, field: string) => {
    setSearchKeyword(keyword);
    setSearchField(field);
  };
  
  // 根据搜索条件过滤设置数据
  const filteredSettingsList = useMemo(() => {
    if (!searchKeyword || !searchField) return settingsList;
    
    return settingsList.filter(setting => {
      const value = setting[searchField as keyof SettingData];
      if (value === null || value === undefined) return false;
      
      return String(value).toLowerCase().includes(searchKeyword.toLowerCase());
    });
  }, [settingsList, searchKeyword, searchField]);

  // 保存设置（普通模式）
  const handleSave = async () => {
    try {
      setLoading(true);
      
      // 提交所有字段，包括各区域脚本
      const response = await axiosInstance.post<ApiResponse>('/setting', {
        region: settings.region,
        instance_type: settings.instance_type,
        disk_size: settings.disk_size,
        password: settings.password,
        script: settings.script,
        jp_script: settings.jp_script,
        sg_script: settings.sg_script
      });
      
      if (response.data.code === 200) {
        addAlert('success', typeof response.data.data === 'string' ? response.data.data : '设置保存成功');
      } else {
        addAlert('error', '保存设置失败');
      }
    } catch (error) {
      addAlert('error', '保存设置请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 打开编辑对话框
  const handleOpenEditDialog = (userId: string) => {
    // 查找对应用户的设置
    const userSettings = settingsList.find(setting => setting.user_id === userId);
    if (userSettings) {
      setEditingSettings({...userSettings});
      setPasswordError('');
      setEditDialogOpen(true);
      setEditScriptTabValue(0); // 重置脚本标签为默认值
    }
  };

  // 关闭编辑对话框
  const handleCloseEditDialog = () => {
    setEditDialogOpen(false);
    setEditingSettings(null);
  };

  // 处理编辑对话框中的输入变化
  const handleEditChange = (field: keyof SettingData) => (event: React.ChangeEvent<HTMLInputElement>) => {
    if (!editingSettings) return;
    
    // 清除之前的错误
    if (field === 'password') {
      setPasswordError('');
    }
    
    setEditingSettings({
      ...editingSettings,
      [field]: event.target.value
    });
  };

  // 验证密码
  const validatePassword = (password: string): boolean => {
    // 要求密码包含大小写字母、数字和特殊字符
    const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$/;
    
    if (!passwordRegex.test(password)) {
      setPasswordError('密码必须包含大小写字母、数字和特殊字符，且长度不少于8位');
      return false;
    }
    
    return true;
  };

  // 保存编辑后的设置（管理员模式）
  const handleSaveEdit = async () => {
    if (!editingSettings) return;
    
    // 验证密码
    if (editingSettings.password && !validatePassword(editingSettings.password)) {
      return;
    }
    
    try {
      setLoading(true);
      
      const response = await axiosInstance.post<ApiResponse>('/setting/admin/update', {
        user_id: editingSettings.user_id,
        region: editingSettings.region,
        instance_type: editingSettings.instance_type,
        disk_size: editingSettings.disk_size,
        password: editingSettings.password,
        script: editingSettings.script,
        jp_script: editingSettings.jp_script,
        sg_script: editingSettings.sg_script
      });
      
      if (response.data.code === 200) {
        addAlert('success', typeof response.data.data === 'object' && response.data.data.message ? response.data.data.message : '设置更新成功');
        // 关闭对话框
        handleCloseEditDialog();
        // 刷新列表数据
        fetchAllSettings();
      } else {
        addAlert('error', '更新设置失败');
      }
    } catch (error) {
      addAlert('error', '更新设置请求失败');
    } finally {
      setLoading(false);
    }
  };

  // 复制文本到剪贴板
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    addAlert('success', '已复制到剪贴板');
  };

  // 创建用于表格展示的列定义
  const columns: Column[] = [
    {
      id: 'user_id',
      label: '用户ID',
      width: '80px',
      sortable: true
    },
    {
      id: 'region',
      label: '默认区域',
      sortable: true
    },
    {
      id: 'instance_type',
      label: '实例规格',
      sortable: true
    },
    {
      id: 'disk_size',
      label: '硬盘大小',
      sortable: true,
      format: (value) => `${value} GB`
    },
    {
      id: 'password',
      label: '开机密码',
      sortable: false,
      format: (value: any) => (
        <Stack direction="row" spacing={1} alignItems="center">
          <Typography variant="body2" 
            sx={{ 
              cursor: 'pointer',
              '&:hover': { textDecoration: 'underline' }
            }}
            onClick={() => copyToClipboard(value as string)}
          >
            {value ? value : '未设置'}
          </Typography>
        </Stack>
      )
    },
    {
      id: 'script',
      label: '开机脚本',
      sortable: false,
      format: (value: any, row?: any) => {
        // 安全检查：确保 row 存在
        if (!row) return value ? '已设置' : '未设置';
        
        const data = row as SettingData;
        // 安全检查：确保 data 中的属性存在
        const hasScript = data && !!data.script;
        const hasJpScript = data && !!data.jp_script;
        const hasSgScript = data && !!data.sg_script;
        
        if (!hasScript && !hasJpScript && !hasSgScript) return '未设置';
        
        const scripts = [];
        if (hasScript) scripts.push('香港');
        if (hasJpScript) scripts.push('日本');
        if (hasSgScript) scripts.push('新加坡');
        
        return scripts.join(', ') + ' 已设置';
      }
    },
    {
      id: 'user_id',
      label: '操作',
      width: '100px',
      format: (value: any) => (
        <Button 
          variant="contained" 
          color="primary" 
          size="small"
          onClick={(e) => {
            e.stopPropagation(); // 阻止事件冒泡
            handleOpenEditDialog(value as string);
          }}
        >
          编辑
        </Button>
      )
    }
  ];

  return (
    <>
      <Helmet>
        <title>{`开机设置 | ${CONFIG.appName}`}</title>
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
            开机参数设置
          </Typography>
          
          {/* 使用AdminModeToggle组件 */}
          <AdminModeToggle visible={isAdmin} />
        </Stack>

        {/* 管理员模式内容 */}
        {isAdmin && isAdminMode ? (
          <>
            {/* 数据表格 */}
            <Box sx={{ width: '100%', mb: 3 }}>
              <Card>
                <CardContent>
                  {/* 说明和搜索框在同一行 */}
                  <Box 
                    sx={{ 
                      display: 'flex', 
                      justifyContent: 'space-between', 
                      alignItems: 'center',
                      mb: 2
                    }}
                  >
                    <Typography variant="body2" color="text.secondary">
                      点击表格行可编辑对应用户的开机设置，点击密码可复制
                    </Typography>
                    
                    {/* 搜索框 */}
                    <TableSearch
                      columns={[
                        { id: 'user_id', label: '用户ID' },
                        { id: 'region', label: '开机区域' },
                        { id: 'instance_type', label: '实例规格' },
                        { id: 'disk_size', label: '硬盘大小' },
                        { id: 'password', label: '开机密码' }
                      ]}
                      onSearch={handleSearch}
                      position="right"
                      width={300}
                      defaultField="user_id"
                    />
                  </Box>
                  
                  <DataTable
                    columns={columns}
                    data={filteredSettingsList}
                    rowKey="user_id"
                    searchable={false}
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
                编辑用户开机设置 (用户ID: {editingSettings?.user_id})
              </DialogTitle>
              <DialogContent>
                <Stack spacing={3} sx={{ mt: 1 }}>
                  {/* 开机区域 */}
                  <TextField
                    label="默认开机区域"
                    select
                    value={editingSettings?.region || '香港'}
                    onChange={handleEditChange('region')}
                    helperText="设置用户的默认开机区域"
                    fullWidth
                  >
                    {regionOptions.map((option) => (
                      <MenuItem key={option.value} value={option.value}>
                        {option.label}
                      </MenuItem>
                    ))}
                  </TextField>

                  {/* 实例规格 */}
                  <TextField
                    label="实例规格"
                    value={editingSettings?.instance_type || ''}
                    onChange={handleEditChange('instance_type')}
                    helperText="设置实例规格，例如: c5n.large"
                    fullWidth
                  />

                  {/* 硬盘大小 */}
                  <TextField
                    label="硬盘大小"
                    value={editingSettings?.disk_size || 0}
                    onChange={handleEditChange('disk_size')}
                    type="number"
                    InputProps={{
                      inputProps: { min: 1 },
                      endAdornment: <InputAdornment position="end">GB</InputAdornment>,
                    }}
                    helperText="设置硬盘大小，单位为GB"
                    fullWidth
                  />

                  {/* 密码 */}
                  <TextField
                    label="开机密码"
                    value={editingSettings?.password || ''}
                    onChange={handleEditChange('password')}
                    placeholder="请输入实例登录密码"
                    error={!!passwordError}
                    helperText={passwordError || "密码必须包含大小写字母、数字和特殊字符"}
                    fullWidth
                  />

                  {/* 开机脚本标签 */}
                  <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                    <Tabs value={editScriptTabValue} onChange={handleEditScriptTabChange} aria-label="脚本区域标签">
                      <Tab label="香港脚本" />
                      <Tab label="日本脚本" />
                      <Tab label="新加坡脚本" />
                    </Tabs>
                  </Box>

                  {/* 香港开机脚本 */}
                  <TabPanel value={editScriptTabValue} index={0}>
                    <TextField
                      label="香港区域开机脚本"
                      value={editingSettings?.script || ''}
                      onChange={handleEditChange('script')}
                      placeholder="请输入香港区域开机执行的脚本"
                      helperText="脚本将在实例启动时执行（请勿超过3行）"
                      multiline
                      rows={6}
                      fullWidth
                    />
                  </TabPanel>

                  {/* 日本开机脚本 */}
                  <TabPanel value={editScriptTabValue} index={1}>
                    <TextField
                      label="日本区域开机脚本"
                      value={editingSettings?.jp_script || ''}
                      onChange={handleEditChange('jp_script')}
                      placeholder="请输入日本区域开机执行的脚本"
                      helperText="如果未设置，将使用香港区域的脚本"
                      multiline
                      rows={6}
                      fullWidth
                    />
                  </TabPanel>

                  {/* 新加坡开机脚本 */}
                  <TabPanel value={editScriptTabValue} index={2}>
                    <TextField
                      label="新加坡区域开机脚本"
                      value={editingSettings?.sg_script || ''}
                      onChange={handleEditChange('sg_script')}
                      placeholder="请输入新加坡区域开机执行的脚本"
                      helperText="如果未设置，将使用香港区域的脚本"
                      multiline
                      rows={6}
                      fullWidth
                    />
                  </TabPanel>
                </Stack>
              </DialogContent>
              <DialogActions>
                <Button onClick={handleCloseEditDialog}>取消</Button>
                <Button 
                  variant="contained" 
                  onClick={handleSaveEdit}
                  disabled={loading || !!passwordError}
                >
                  保存更改
                </Button>
              </DialogActions>
            </Dialog>
            
            {/* 管理员模式提示 */}
            <Typography variant="body2" color="primary" sx={{ mt: 2 }}>
              当前处于管理员模式，可管理所有用户的开机设置
            </Typography>
          </>
        ) : (
          /* 普通模式内容 */
          <>
            <Box sx={{ width: '100%', mb: 3 }}>
              <Card>
                <CardContent>
                  <Stack spacing={3} sx={{ maxWidth: 600 }}>
                    {/* 开机区域 - 改为可选择 */}
                    <TextField
                      label="默认开机区域"
                      select
                      value={settings.region}
                      onChange={handleChange('region')}
                      helperText="设置开机默认区域，可在实例管理中指定其他区域"
                      sx={{ "& .MuiFormHelperText-root": { fontSize: "0.875rem" } }}
                      fullWidth
                    >
                      {regionOptions.map((option) => (
                        <MenuItem key={option.value} value={option.value}>
                          {option.label}
                        </MenuItem>
                      ))}
                    </TextField>

                    {/* 只读字段：实例规格 */}
                    <TextField
                      label="实例规格"
                      value={settings.instance_type}
                      InputProps={{
                        readOnly: true,
                      }}
                      helperText="系统默认设置，不可修改"
                      fullWidth
                    />

                    {/* 只读字段：硬盘大小 */}
                    <TextField
                      label="硬盘大小"
                      value={settings.disk_size}
                      InputProps={{
                        readOnly: true,
                        endAdornment: <InputAdornment position="end">GB</InputAdornment>,
                      }}
                      helperText="系统默认设置，不可修改"
                      fullWidth
                    />

                    {/* 可编辑字段：密码 */}
                    <TextField
                      label="开机密码"
                      value={settings.password}
                      onChange={handleChange('password')}
                      placeholder="请输入实例登录密码"
                      helperText="密码必须包含大小写字母、数字和特殊字符"
                      sx={{ "& .MuiFormHelperText-root": { fontSize: "0.875rem" } }}
                      fullWidth
                    />

                    {/* 脚本标签 */}
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                      <Tabs value={scriptTabValue} onChange={handleScriptTabChange} aria-label="脚本区域标签">
                        <Tab label="香港脚本" />
                        <Tab label="日本脚本" />
                        <Tab label="新加坡脚本" />
                      </Tabs>
                    </Box>

                    {/* 香港开机脚本 */}
                    <TabPanel value={scriptTabValue} index={0}>
                      <TextField
                        label="香港区域开机脚本"
                        value={settings.script}
                        onChange={handleChange('script')}
                        placeholder="请输入香港区域开机执行的脚本"
                        helperText="脚本将在实例启动时执行（请勿超过3行）"
                        sx={{ "& .MuiFormHelperText-root": { fontSize: "0.875rem" } }}
                        multiline
                        rows={6}
                        fullWidth
                      />
                    </TabPanel>

                    {/* 日本开机脚本 */}
                    <TabPanel value={scriptTabValue} index={1}>
                      <TextField
                        label="日本区域开机脚本"
                        value={settings.jp_script}
                        onChange={handleChange('jp_script')}
                        placeholder="请输入日本区域开机执行的脚本"
                        helperText="如果未设置，将使用香港区域的脚本"
                        sx={{ "& .MuiFormHelperText-root": { fontSize: "0.875rem" } }}
                        multiline
                        rows={6}
                        fullWidth
                      />
                    </TabPanel>

                    {/* 新加坡开机脚本 */}
                    <TabPanel value={scriptTabValue} index={2}>
                      <TextField
                        label="新加坡区域开机脚本"
                        value={settings.sg_script}
                        onChange={handleChange('sg_script')}
                        placeholder="请输入新加坡区域开机执行的脚本"
                        helperText="如果未设置，将使用香港区域的脚本"
                        sx={{ "& .MuiFormHelperText-root": { fontSize: "0.875rem" } }}
                        multiline
                        rows={6}
                        fullWidth
                      />
                    </TabPanel>
                  </Stack>
                </CardContent>
              </Card>
            </Box>

            {/* 保存按钮 */}
            <Box sx={{ display: 'flex', justifyContent: 'flex-start' }}>
              <CustomButton 
                onClick={handleSave} 
                disabled={loading}
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