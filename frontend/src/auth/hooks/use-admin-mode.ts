// src/auth/hooks/use-admin-mode.ts
import { useState, useEffect, useCallback } from 'react';
import { useIsAdmin } from './use-is-admin'; // 新增导入

// localStorage 键名，用于存储管理员模式状态
const ADMIN_MODE_STORAGE_KEY = 'adminModeEnabled';

/**
 * 自定义钩子，用于管理管理员模式状态
 * @returns 管理员模式状态和切换函数
 */
export function useAdminMode() {
  const isAdmin = useIsAdmin(); // 新增：获取当前用户是否为管理员
  
  // 从 localStorage 初始化管理员模式状态
  const [isAdminMode, setIsAdminMode] = useState<boolean>(() => {
    try {
      const savedMode = localStorage.getItem(ADMIN_MODE_STORAGE_KEY);
      return savedMode === 'true';
    } catch (error) {
      console.error('从localStorage读取管理员模式状态失败:', error);
      return false;
    }
  });

  // 新增：当用户状态变化时检查
  useEffect(() => {
    if (!isAdmin && isAdminMode) {
      // 非管理员用户不能使用管理员模式，强制关闭
      localStorage.removeItem(ADMIN_MODE_STORAGE_KEY);
      setIsAdminMode(false);
    }
  }, [isAdmin, isAdminMode]);

  // 创建一个自定义事件，用于同步当前页面内的状态
  const createCustomEvent = (value: boolean) => {
    return new CustomEvent('adminModeChange', { 
      detail: { value },
      bubbles: true 
    });
  };

  // 监听自定义事件和存储事件
  useEffect(() => {
    // 处理自定义事件，用于同步当前页面内的状态
    const handleCustomEvent = (event: Event) => {
      const customEvent = event as CustomEvent<{value: boolean}>;
      setIsAdminMode(customEvent.detail.value);
    };

    // 处理存储事件，当其他页面更改了localStorage时同步状态
    const handleStorageChange = (event: StorageEvent) => {
      if (event.key === ADMIN_MODE_STORAGE_KEY) {
        setIsAdminMode(event.newValue === 'true');
      }
    };

    // 添加事件监听
    window.addEventListener('adminModeChange', handleCustomEvent);
    window.addEventListener('storage', handleStorageChange);

    // 组件卸载时移除事件监听
    return () => {
      window.removeEventListener('adminModeChange', handleCustomEvent);
      window.removeEventListener('storage', handleStorageChange);
    };
  }, []);

  /**
   * 切换管理员模式状态
   * @param mode 可选，指定新的模式，如果不提供则切换当前状态
   * @returns 新的管理员模式状态
   */
  const toggleAdminMode = useCallback((mode?: boolean) => {
    const newMode = mode !== undefined ? mode : !isAdminMode;
    
    try {
      // 更新localStorage
      localStorage.setItem(ADMIN_MODE_STORAGE_KEY, String(newMode));
      
      // 分发自定义事件，通知同一页面内的其他组件
      window.dispatchEvent(createCustomEvent(newMode));
      
      // 更新本地状态
      setIsAdminMode(newMode);
    } catch (error) {
      console.error('保存管理员模式状态失败:', error);
    }
    
    return newMode;
  }, [isAdminMode]);

  // 修改返回值，确保非管理员用户始终返回false
  return { 
    isAdminMode: isAdmin ? isAdminMode : false, 
    toggleAdminMode 
  };
}