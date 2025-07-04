// src\auth\context\jwt\utils.ts
import axios from 'src/lib/axios';
import { TOKEN_STORAGE_KEY, USER_STORAGE_KEY } from './constant';
import { UserType } from '../../types';

// ----------------------------------------------------------------------

export function setSession(
  accessToken: string | null, 
  userData?: { 
    user_id: string; 
    is_admin: number;
    email?: string; // 添加可选的email字段
  }
) {
  try {
    if (accessToken) {
      localStorage.setItem(TOKEN_STORAGE_KEY, accessToken);
      axios.defaults.headers.common.Authorization = `Bearer ${accessToken}`;
      
      // Store user data if provided
      if (userData) {
        const user: UserType = {
          id: userData.user_id, // 保持字符串类型
          email: userData.email || '', // 使用后端返回的email，如果不存在则使用空字符串
          isAdmin: userData.is_admin,
          role: userData.is_admin === 1 ? 'admin' : 'user' // 设置默认role
        };
        localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(user));
      }
    } else {
      localStorage.removeItem(TOKEN_STORAGE_KEY);
      localStorage.removeItem(USER_STORAGE_KEY);
      delete axios.defaults.headers.common.Authorization;
    }
  } catch (error) {
    console.error('Error during set session:', error);
    throw error;
  }
}

export function getStoredUser(): UserType {
  try {
    const storedUser = localStorage.getItem(USER_STORAGE_KEY);
    return storedUser ? JSON.parse(storedUser) : null;
  } catch {
    return null;
  }
}