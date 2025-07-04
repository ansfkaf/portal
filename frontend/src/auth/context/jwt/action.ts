// src/auth/context/jwt/action.ts
import axios, { endpoints } from 'src/lib/axios';
import { setSession } from './utils';

export type SignInParams = {
  email: string;
  password: string;
};

export type SignUpParams = {
  email: string;
  password: string;
};

export type SignInResponse = {
  data: {
    token: string;
    user_id: string;
    is_admin: number;
    email: string; // 添加email字段
  };
  code: number;
  message: string;
};

export const signInWithPassword = async ({ email, password }: SignInParams): Promise<void> => {
  try {
    const res = await axios.post(endpoints.auth.login, { email, password });
    const { token, user_id, is_admin, email: userEmail } = res.data.data;

    if (!token) {
      throw new Error('Access token not found in response');
    }

    setSession(token, { user_id, is_admin, email: userEmail });
  } catch (error: any) {
    console.error('Error during sign in:', error);
    throw new Error(error.response?.data?.message || error.message || 'Failed to sign in');
  }
};

export const signUp = async ({ email, password }: SignUpParams): Promise<void> => {
  try {
    const res = await axios.post(endpoints.auth.register, {
      email,
      password,
    });
    
    const { token, user_id, is_admin, email: userEmail } = res.data.data;

    if (!token) {
      throw new Error('Access token not found in response');
    }

    setSession(token, { user_id, is_admin, email: userEmail });
  } catch (error: any) {
    // 直接抛出后端返回的错误消息
    if (error.response?.data?.message) {
      throw new Error(error.response.data.message);
    }
    // 如果没有具体错误消息，才使用默认消息
    throw new Error('Failed to sign up');
  }
};

export const signOut = async (): Promise<void> => {
  try {
    setSession(null);
    
    // 添加：清除管理员模式状态
    localStorage.removeItem('adminModeEnabled');
  } catch (error) {
    console.error('Error during sign out:', error);
    throw error;
  }
};