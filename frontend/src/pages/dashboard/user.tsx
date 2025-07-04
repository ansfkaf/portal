// src/pages/dashboard/user.tsx
import { useEffect, useState, useMemo } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import { DataTable, type Column } from 'src/components/table/data-table';
import { TableSearch } from 'src/components/table/table-search';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import axiosInstance from 'src/lib/axios';
import { 
  Dialog, 
  DialogContent, 
  DialogTitle, 
  IconButton, 
  TextField,
  DialogActions,
  FormControl,
  FormControlLabel,
  Checkbox
} from '@mui/material';

// 接口类型定义
interface UserData {
  id: string;
  email: string;
  is_admin: number; // 1表示管理员，0表示普通用户
}

interface UserResponse {
  code: number;
  message: string;
  data: {
    total: number;
    users: UserData[];
  };
}

interface MakeupResponse {
  code: number;
  message: string;
  data: {
    failed_ids: string[];
    message: string;
    success_count: number;
  };
}

interface CreateUserResponse {
  code: number;
  message: string;
  data: {
    message: string;
    user: UserData;
  } | null;
}

export default function UserPage() {
  const [users, setUsers] = useState<UserData[]>([]);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [editPassword, setEditPassword] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const [isAdmin, setIsAdmin] = useState(false);
  const [newUserEmail, setNewUserEmail] = useState('');
  const [newUserPassword, setNewUserPassword] = useState('Aa112233');
  const [newUserIsAdmin, setNewUserIsAdmin] = useState(false);
  
  // 搜索状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchField, setSearchField] = useState('id'); // 默认按用户ID搜索
  
  const { addAlert } = useAlert();

  // 获取用户列表数据
  const fetchUsers = async () => {
    try {
      const response = await axiosInstance.post<UserResponse>('/user/get');
      if (response.data.code === 200) {
        const sortedData = response.data.data.users.sort((a: UserData, b: UserData) => 
          parseInt(a.id) - parseInt(b.id)
        );
        setUsers(sortedData);
      } else {
        addAlert('error', '获取数据失败');
      }
    } catch (error) {
      addAlert('error', '获取用户列表失败');
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  // 开机功能
  const handlePowerOn = async () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要补机的用户');
      return;
    }

    try {
      const response = await axiosInstance.post<MakeupResponse>('/user/makeup', {
        ids: selectedIds
      });
      
      if (response.data.code === 200) {
        addAlert('success', response.data.data.message || '补机任务已提交');
      } else {
        addAlert('error', '补机失败');
      }
    } catch (error) {
      addAlert('error', '补机请求失败');
    }
  };

  // 重置密码
  const handleResetPassword = () => {
    if (selectedIds.length === 0) {
      addAlert('warning', '请选择要重置密码的用户');
      return;
    }
    
    // 打开编辑对话框，默认只修改密码
    setEditPassword('');
    setEditEmail('');
    setIsAdmin(false);
    setIsEditDialogOpen(true);
  };

  // 编辑用户
  const handleEdit = (userId: string) => {
    setSelectedIds([userId]);
    const user = users.find(u => u.id === userId);
    if (user) {
      setIsAdmin(user.is_admin === 1);
      setEditEmail(user.email);
    }
    setEditPassword('');
    setIsEditDialogOpen(true);
  };

  // 处理搜索
  const handleSearch = (keyword: string, field: string) => {
    setSearchKeyword(keyword);
    setSearchField(field);
  };
  
  // 根据搜索条件过滤用户数据
  const filteredUsers = useMemo(() => {
    if (!searchKeyword || !searchField) return users;
    
    return users.filter(user => {
      const value = user[searchField as keyof UserData];
      if (value === null || value === undefined) return false;
      
      return String(value).toLowerCase().includes(searchKeyword.toLowerCase());
    });
  }, [users, searchKeyword, searchField]);

  // 提交编辑
  const handleSubmitEdit = async () => {
    try {
      const updateData: {
        ids: string[];
        email?: string;
        password?: string;
        is_admin: number;
      } = {
        ids: selectedIds,
        is_admin: isAdmin ? 1 : 0
      };
      
      if (editPassword) {
        updateData.password = editPassword;
      }
      
      if (editEmail) {
        updateData.email = editEmail;
      }
      
      const response = await axiosInstance.post('/user/update', updateData);
      
      if (response.data.code === 200) {
        addAlert('success', '用户信息更新成功');
        setIsEditDialogOpen(false);
        fetchUsers(); // 重新加载数据
      } else {
        addAlert('error', '更新失败');
      }
    } catch (error) {
      addAlert('error', '更新请求失败');
    }
  };

  // 打开创建用户对话框
  const handleOpenCreateDialog = () => {
    setNewUserEmail('');
    setNewUserPassword('Aa112233');
    setNewUserIsAdmin(false);
    setIsCreateDialogOpen(true);
  };

  // 提交创建用户
  const handleCreateUser = async () => {
    if (!newUserEmail || !newUserPassword) {
      addAlert('warning', '邮箱和密码不能为空');
      return;
    }

    try {
      const response = await axiosInstance.post<CreateUserResponse>('/user/create', {
        email: newUserEmail,
        password: newUserPassword,
        is_admin: newUserIsAdmin ? 1 : 0
      });
      
      if (response.data.code === 200) {
        addAlert('success', response.data.data?.message || '用户创建成功');
        setIsCreateDialogOpen(false);
        fetchUsers(); // 重新加载数据
      } else {
        addAlert('error', response.data.message || '创建用户失败');
      }
    } catch (error: any) {
      addAlert('error', error.response?.data?.message || '创建用户请求失败');
    }
  };

  // 由于format函数只接受一个参数，我们需要修改我们的方法
  // 为每个用户添加actions属性
  const usersWithActions = filteredUsers.map(user => ({
    ...user,
    actions: user.id // 将id作为actions的值，后面会用这个值来找到正确的用户
  }));

  // 主表格列定义
  const columns: Column[] = [
    { 
      id: 'id', 
      label: '用户ID', 
      sortable: true,
      width: '100px'
    },
    { 
      id: 'email', 
      label: '邮箱', 
      sortable: true
    },
    { 
      id: 'is_admin', 
      label: '管理员',
      sortable: true,
      align: 'center',
      format: (value: any) => value === 1 ? '是' : '否'
    },
    {
      id: 'actions',
      label: '操作',
      align: 'center',
      format: (userId: string) => (
        <CustomButton 
          onClick={(e: React.MouseEvent) => {
            e.stopPropagation();
            handleEdit(userId);
          }}
          size="small"
        >
          编辑
        </CustomButton>
      )
    }
  ];

  return (
    <>
      <Helmet>
        <title>{`用户管理 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
          用户管理
        </Typography>

        {/* 按钮组和搜索框在同一行 */}
        <Box sx={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          mb: 3 
        }}>
          {/* 按钮组 */}
          <Stack direction="row" spacing={2}>
            <CustomButton onClick={handlePowerOn}>
              开机
            </CustomButton>
            <CustomButton onClick={handleResetPassword}>
              重置密码
            </CustomButton>
            <CustomButton onClick={handleOpenCreateDialog}>
              新增用户
            </CustomButton>
          </Stack>
          
          {/* 搜索框 */}
          <TableSearch
            columns={[
              { id: 'id', label: '用户ID' },
              { id: 'email', label: '邮箱' },
              { id: 'is_admin', label: '管理员' }
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
            data={usersWithActions}
            selectable
            onSelectionChange={setSelectedIds}
            searchable={false}
          />
        </Box>

        {/* 编辑用户对话框 */}
        <Dialog
          open={isEditDialogOpen}
          onClose={() => setIsEditDialogOpen(false)}
          maxWidth="sm"
          fullWidth
        >
          <DialogTitle sx={{ m: 0, p: 2 }}>
            {selectedIds.length > 1 ? '批量编辑用户' : '编辑用户'}
            <IconButton
              onClick={() => setIsEditDialogOpen(false)}
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
            <Box sx={{ pt: 2 }}>
              <TextField
                fullWidth
                label="邮箱"
                type="email"
                variant="outlined"
                value={editEmail}
                onChange={(e) => setEditEmail(e.target.value)}
                margin="normal"
                placeholder={selectedIds.length > 1 ? "批量编辑时留空表示不修改邮箱" : ""}
              />
              
              <TextField
                fullWidth
                label="密码"
                type="text"
                variant="outlined"
                value={editPassword}
                onChange={(e) => setEditPassword(e.target.value)}
                placeholder="留空表示不修改密码"
                margin="normal"
              />
              
              <FormControl component="fieldset" sx={{ mt: 2 }}>
                <FormControlLabel
                  control={
                    <Checkbox 
                      checked={isAdmin} 
                      onChange={(e) => setIsAdmin(e.target.checked)}
                    />
                  }
                  label="设为管理员"
                />
              </FormControl>
            </Box>
          </DialogContent>
          <DialogActions>
            <CustomButton 
              onClick={() => setIsEditDialogOpen(false)}
              color="inherit"
            >
              取消
            </CustomButton>
            <CustomButton onClick={handleSubmitEdit}>
              保存
            </CustomButton>
          </DialogActions>
        </Dialog>

        {/* 新增用户对话框 */}
        <Dialog
          open={isCreateDialogOpen}
          onClose={() => setIsCreateDialogOpen(false)}
          maxWidth="sm"
          fullWidth
        >
          <DialogTitle sx={{ m: 0, p: 2 }}>
            新增用户
            <IconButton
              onClick={() => setIsCreateDialogOpen(false)}
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
            <Box sx={{ pt: 2 }}>
              <TextField
                fullWidth
                label="邮箱"
                type="email"
                variant="outlined"
                value={newUserEmail}
                onChange={(e) => setNewUserEmail(e.target.value)}
                margin="normal"
                required
              />
              
              <TextField
                fullWidth
                label="密码"
                type="password"
                variant="outlined"
                value={newUserPassword}
                onChange={(e) => setNewUserPassword(e.target.value)}
                margin="normal"
                required
              />
              
              <FormControl component="fieldset" sx={{ mt: 2 }}>
                <FormControlLabel
                  control={
                    <Checkbox 
                      checked={newUserIsAdmin} 
                      onChange={(e) => setNewUserIsAdmin(e.target.checked)}
                    />
                  }
                  label="设为管理员"
                />
              </FormControl>
            </Box>
          </DialogContent>
          <DialogActions>
            <CustomButton 
              onClick={() => setIsCreateDialogOpen(false)}
              color="inherit"
            >
              取消
            </CustomButton>
            <CustomButton onClick={handleCreateUser}>
              创建
            </CustomButton>
          </DialogActions>
        </Dialog>
      </DashboardContent>
    </>
  );
}