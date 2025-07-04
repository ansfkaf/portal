// src/pages/dashboard/import.tsx
import { useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import { CustomButton } from 'src/components/custom-button/custom-button';
import { useAlert } from 'src/components/custom-alert/custom-alert';
import TextField from '@mui/material/TextField';
import Paper from '@mui/material/Paper';
import Divider from '@mui/material/Divider';
import axiosInstance from 'src/lib/axios';

// 接口类型定义
interface ImportResponse {
  code: number;
  message: string;
  data: {
    summary: {
      success_count: number;
      failed_count: number;
      duplicate_count: number;
      format_error_count: number;
    };
    details: {
      duplicate_list: string[];
      format_error_list: string[];
    };
  };
}

export default function ImportPage() {
  const [content, setContent] = useState('');
  const [showResult, setShowResult] = useState(false);
  const [result, setResult] = useState<ImportResponse['data'] | null>(null);
  const { addAlert } = useAlert();

  // 处理提交
  const handleSubmit = async () => {
    if (!content.trim()) {
      addAlert('warning', '请输入要导入的账号信息');
      return;
    }

    try {
      const response = await axiosInstance.post<ImportResponse>('/import', {
        content: content
      });

      if (response.data.code === 200) {
        setResult(response.data.data);
        setShowResult(true);
        addAlert('success', '提交成功');
      } else {
        addAlert('error', '提交失败');
        setShowResult(false);
      }
    } catch (error) {
      addAlert('error', '提交失败');
      setShowResult(false);
    }
  };

  return (
    <>
      <Helmet>
        <title>{`账户导入 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
          账户导入
        </Typography>

        {/* 导入说明 */}
        <Typography variant="body1" sx={{ mb: 2 }}>
          请按照以下格式输入账号信息，每行一个账号：
        </Typography>
        <Typography variant="body2" sx={{ mb: 3, color: 'text.secondary' }}>
          账号----密码----key1----key2
        </Typography>

        {/* 文本输入框 */}
        <TextField
          multiline
          rows={10}
          fullWidth
          placeholder="请输入账号信息..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
          sx={{ mb: 3 }}
        />

        {/* 提交按钮 */}
        <Box sx={{ mb: 3 }}>
          <CustomButton onClick={handleSubmit}>
            提交
          </CustomButton>
        </Box>

        {/* 结果显示区 */}
        {showResult && result && (
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              导入结果统计
            </Typography>
            
            {/* 统计数据 */}
            <Box sx={{ display: 'flex', gap: 4, mb: 3 }}>
              {result.summary.success_count > 0 && (
                <Typography color="success.main">
                  导入成功：{result.summary.success_count} 个
                </Typography>
              )}
              {result.summary.failed_count > 0 && (
                <Typography color="error.main">
                  导入失败：{result.summary.failed_count} 个
                </Typography>
              )}
              {result.summary.duplicate_count > 0 && (
                <Typography color="warning.main">
                  重复账号：{result.summary.duplicate_count} 个
                </Typography>
              )}
              {result.summary.format_error_count > 0 && (
                <Typography color="error.main">
                  格式错误：{result.summary.format_error_count} 个
                </Typography>
              )}
            </Box>

            {/* 分割线 */}
            <Divider sx={{ my: 2 }} />

            {/* 详细信息 */}
            <Typography variant="h6" sx={{ mb: 2 }}>
              详细信息
            </Typography>
            
            {/* 重复账号列表 */}
            {result.details?.duplicate_list?.length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle1" color="warning.main" sx={{ mb: 1 }}>
                  重复的账号：
                </Typography>
                {result.details.duplicate_list.map((item, index) => (
                  <Typography key={index} variant="body2" sx={{ ml: 2 }}>
                    {item}
                  </Typography>
                ))}
              </Box>
            )}

            {/* 格式错误列表 */}
            {result.details?.format_error_list?.length > 0 && (
              <Box>
                <Typography variant="subtitle1" color="error.main" sx={{ mb: 1 }}>
                  格式错误的账号：
                </Typography>
                {result.details.format_error_list.map((item, index) => (
                  <Typography key={index} variant="body2" sx={{ ml: 2 }}>
                    {item}
                  </Typography>
                ))}
              </Box>
            )}
          </Paper>
        )}
      </DashboardContent>
    </>
  );
}