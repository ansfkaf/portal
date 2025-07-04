// src/pages/dashboard/database.tsx
import { useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import TextField from '@mui/material/TextField';
import Button from '@mui/material/Button';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import axiosInstance from 'src/lib/axios';

// API响应接口
interface ApiResponse<T = any> {
  success: boolean;
  message: string;
  data: T;
}

export default function DatabasePage() {
  // 备份文件路径状态
  const [backupFilePath, setBackupFilePath] = useState<string>('');
  
  // 加载状态
  const [loadingBackup, setLoadingBackup] = useState(false);
  const [loadingRestore, setLoadingRestore] = useState(false);
  
  // 使用提示组件
  const { addAlert } = useAlert();

  // 备份数据库
  const handleBackup = async () => {
    try {
      setLoadingBackup(true);
      
      const response = await axiosInstance.post<ApiResponse>('/admin/backup');
      
      if (response.data.success) {
        addAlert('success', response.data.message || '数据库备份成功');
        // 如果需要，可以设置备份文件路径到输入框
        if (response.data.data && response.data.data.s3Path) {
          setBackupFilePath(response.data.data.s3Path);
        }
      } else {
        addAlert('error', response.data.message || '数据库备份失败');
      }
    } catch (error) {
      addAlert('error', '备份数据库请求失败');
    } finally {
      setLoadingBackup(false);
    }
  };

  // 恢复数据库
  const handleRestore = async () => {
    if (!backupFilePath.trim()) {
      addAlert('error', '请输入备份文件路径');
      return;
    }
    
    try {
      setLoadingRestore(true);
      
      const response = await axiosInstance.post<ApiResponse>('/admin/restore', {
        backupFilePath: backupFilePath.trim()
      });
      
      if (response.data.success) {
        addAlert('success', response.data.message || '数据库恢复成功');
      } else {
        addAlert('error', response.data.message || '数据库恢复失败');
      }
    } catch (error) {
      addAlert('error', '恢复数据库请求失败');
    } finally {
      setLoadingRestore(false);
    }
  };

  return (
    <>
      <Helmet>
        <title>{`数据库管理 | ${CONFIG.appName}`}</title>
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
            数据库管理
          </Typography>
        </Stack>

        {/* 数据库管理卡片 */}
        <Box sx={{ width: '100%', mb: 3 }}>
          <Card>
            <CardContent>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
                在这里可以进行数据库备份和恢复操作，请谨慎使用恢复功能，以免数据丢失。
              </Typography>
              
              <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* 备份文件路径输入 */}
                <TextField
                  label="备份文件路径"
                  value={backupFilePath}
                  onChange={(e) => setBackupFilePath(e.target.value)}
                  placeholder="输入需要恢复的备份文件路径"
                  helperText="例如: /root/portal_20250311_140504.sql"
                  sx={{ "& .MuiFormHelperText-root": { fontSize: "0.875rem" } }}
                  fullWidth
                />

                {/* 操作按钮 */}
                <Stack direction="row" spacing={2}>
                  <CustomButton
                    onClick={handleBackup}
                    disabled={loadingBackup}
                  >
                    备份数据库
                  </CustomButton>
                  
                  <Button
                    variant="contained"
                    color="warning"
                    onClick={handleRestore}
                    disabled={loadingRestore || !backupFilePath.trim()}
                  >
                    从备份恢复
                  </Button>
                </Stack>
              </Stack>
            </CardContent>
          </Card>
        </Box>

        {/* 警告提示 */}
        <Typography variant="body2" color="error" sx={{ mt: 2 }}>
          注意：恢复操作将覆盖当前数据库中的所有数据，请确保已有备份后再进行操作。
        </Typography>
      </DashboardContent>
    </>
  );
}