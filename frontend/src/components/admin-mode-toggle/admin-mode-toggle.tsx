// src/components/admin-mode-toggle/admin-mode-toggle.tsx
import React, { useEffect } from 'react';
import FormControlLabel from '@mui/material/FormControlLabel';
import Switch from '@mui/material/Switch';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useAdminMode } from 'src/auth/hooks/use-admin-mode';

// 组件属性接口
interface AdminModeToggleProps {
  // 是否显示组件（通常根据用户是否为管理员决定）
  visible?: boolean;
  
  // 自定义回调函数，当模式变化时调用
  onModeChange?: (isAdminMode: boolean) => void;

  // 自定义样式
  sx?: React.CSSProperties | object;
  
  // 自定义标签
  adminLabel?: string;
  normalLabel?: string;
  
  // 显示备注文本
  showHint?: boolean;
  
  // 自定义备注文本
  adminHint?: string;
  normalHint?: string;
}

/**
 * 管理员模式切换组件
 * 提供管理员模式和普通模式的切换功能，并保存状态到localStorage
 */
export function AdminModeToggle({
  visible = true,
  onModeChange,
  sx = {},
  adminLabel = "管理员模式",
  normalLabel = "普通模式",
  showHint = false,
  adminHint = "当前处于管理员模式",
  normalHint = "当前处于普通模式"
}: AdminModeToggleProps) {
  // 使用自定义钩子管理状态
  const { isAdminMode, toggleAdminMode } = useAdminMode();

  // 监听管理员模式变化，当状态改变时调用回调
  useEffect(() => {
    // 这个useEffect会在isAdminMode变化时触发
    if (onModeChange) {
      onModeChange(isAdminMode);
    }
  }, [isAdminMode, onModeChange]);

  // 如果不可见，则返回null
  if (!visible) {
    return null;
  }

  // 处理模式切换
  const handleToggle = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newMode = event.target.checked;
    toggleAdminMode(newMode);
    // 注意: 不需要在这里调用onModeChange，因为上面的useEffect会处理
  };

  return (
    <Box sx={{ ...sx }}>
      <FormControlLabel
        control={
          <Switch
            checked={isAdminMode}
            onChange={handleToggle}
            color="primary"
          />
        }
        label={isAdminMode ? adminLabel : normalLabel}
      />
      
      {/* 可选的提示文本 */}
      {showHint && (
        <Typography variant="body2" color="primary" sx={{ mt: 1, fontSize: '0.75rem' }}>
          {isAdminMode ? adminHint : normalHint}
        </Typography>
      )}
    </Box>
  );
}

// 导出默认组件
export default AdminModeToggle;