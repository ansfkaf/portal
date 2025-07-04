// src/pages/dashboard/home.tsx
import { Helmet } from 'react-helmet-async';
import { CONFIG } from 'src/global-config';
import { DashboardContent } from 'src/layouts/dashboard';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid2';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Divider from '@mui/material/Divider';

export default function HomePage() {
  return (
    <>
      <Helmet>
        <title>{`首页 | ${CONFIG.appName}`}</title>
      </Helmet>

      <DashboardContent maxWidth="xl">
        {/* 标题部分 */}
        <Typography variant="h4" sx={{ mb: 3, fontWeight: 'bold' }}>
          欢迎使用AWS管理面板 👋
        </Typography>

        {/* 内容部分 */}
        <Paper elevation={2} sx={{ p: 3, mb: 4 }}>
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              售后联系：
            </Typography>
            <Link 
              href="https://t.me/aws007_cc" 
              target="_blank" 
              rel="noopener"
              sx={{ ml: 2 }}
            >
              https://t.me/aws007_cc
            </Link>
          </Box>
          
          <Divider sx={{ my: 2 }} />
          
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              使用教程：
            </Typography>
            <Typography variant="body2" sx={{ ml: 2 }}>
              在开机设置里修改默认开机密码，添加开机脚本。
            </Typography>
          </Box>
          
          <Divider sx={{ my: 2 }} />
          
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              在线实例管理：
            </Typography>
            <Typography variant="body2" sx={{ ml: 2 }}>
              可以更换实例IP，删除实例。
            </Typography>
          </Box>
          
          <Divider sx={{ my: 2 }} />
          
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'medium' }}>
              监控模块新功能：
            </Typography>
            <Box sx={{ ml: 2 }}>
              <Typography variant="body2" sx={{ mb: 1 }}>
                • Telegram 通知：启用后，可以在bot接收实例离线、上线的通知
              </Typography>
              <Typography variant="body2">
                • IP范围限制：可以自动刷指定的ip段
              </Typography>
            </Box>
          </Box>
          
          <Divider sx={{ my: 2 }} />
          
          <Box>
            <Typography variant="body2" sx={{ color: 'text.secondary', fontStyle: 'italic' }}>
              更多功能开发中...
            </Typography>
          </Box>
        </Paper>
      </DashboardContent>
    </>
  );
}