// src/components/custom-button/custom-button.tsx
import { styled } from '@mui/material/styles';
import Button, { ButtonProps } from '@mui/material/Button';

interface CustomButtonProps extends ButtonProps {
  bgColor?: string;  // 背景颜色
  hoverColor?: string;  // 悬停颜色
}

// 基础小按钮
const BaseButton = styled(Button)<CustomButtonProps>(({ bgColor = '#1976d2', hoverColor = '#1565c0' }) => ({
  backgroundColor: bgColor,
  color: 'white',
  width: '100px',
  height: '42px',
  fontSize: '15px',
  fontWeight: 400,
  '&:hover': {
    backgroundColor: hoverColor,
  },
}));

export const CustomButton = BaseButton;