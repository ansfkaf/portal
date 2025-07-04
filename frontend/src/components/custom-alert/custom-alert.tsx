// src/components/custom-alert/custom-alert.tsx
import React, { createContext, useContext, useState, useCallback } from 'react';
import { Alert, Box } from '@mui/material';

// 类型定义
type AlertType = 'success' | 'error' | 'warning' | 'info';

interface AlertMessage {
  id: string;
  type: AlertType;
  message: string;
}

interface AlertOptions {
  duration?: number; // 持续时间，单位毫秒
}

interface AlertContextType {
  alerts: AlertMessage[];
  addAlert: (type: AlertType, message: string, options?: AlertOptions) => void;
  removeAlert: (id: string) => void;
}

// 创建 Context
const AlertContext = createContext<AlertContextType | undefined>(undefined);

// Alert Provider 组件
export const AlertProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [alerts, setAlerts] = useState<AlertMessage[]>([]);

  const removeAlert = useCallback((id: string) => {
    setAlerts(prev => prev.filter(alert => alert.id !== id));
  }, []);

  const addAlert = useCallback((type: AlertType, message: string, options: AlertOptions = {}) => {
    const id = Math.random().toString(36).substring(7);
    const duration = options.duration || 3000; // 默认3秒后消失

    setAlerts(prev => [...prev, { id, type, message }]);

    // 设置自动移除
    setTimeout(() => {
      removeAlert(id);
    }, duration);
  }, [removeAlert]);

  return (
    <AlertContext.Provider value={{ alerts, addAlert, removeAlert }}>
      <AlertContainer />
      {children}
    </AlertContext.Provider>
  );
};

// Alert 容器组件
const AlertContainer: React.FC = () => {
  const { alerts } = useAlert();

  return (
    <Box
      sx={{
        position: 'fixed',
        top: 65, // 调整为标题下方位置
        left: '50%',
        transform: 'translateX(-50%)',
        zIndex: 1101, // 确保在其他元素之上
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
        width: 'auto',
        minWidth: '200px',
        maxWidth: '400px',
        padding: '0 16px',
      }}
    >
      {alerts.map(alert => (
        <Alert
          key={alert.id}
          severity={alert.type}
          sx={{
            width: '100%',
            boxShadow: 2,
            '& .MuiAlert-message': {
              width: '100%',
            },
          }}
        >
          {alert.message}
        </Alert>
      ))}
    </Box>
  );
};

// Hook 用于在组件中使用 Alert
export const useAlert = () => {
  const context = useContext(AlertContext);
  if (context === undefined) {
    throw new Error('useAlert 必须在 AlertProvider 内部使用');
  }
  return context;
};