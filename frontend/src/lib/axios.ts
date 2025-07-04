// src/lib/axios.ts
import type { AxiosRequestConfig } from 'axios';
import axios from 'axios';
import { paths } from 'src/routes/paths';

// ----------------------------------------------------------------------

const axiosInstance = axios.create({ 
  baseURL: '/api', // 修改为使用相对路径，配合 Vite 代理
  timeout: 100000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for adding auth token
axiosInstance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor
axiosInstance.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && !error.config.url.includes('/login')) {
      localStorage.removeItem('token');
      localStorage.removeItem('user_data');
      window.location.href = paths.auth.jwt.signIn;
      return Promise.reject(new Error('Unauthorized access'));
    }
    // 直接返回错误响应
    return Promise.reject(error);
  }
);

export default axiosInstance;

export const fetcher = async (args: string | [string, AxiosRequestConfig]) => {
  try {
    const [url, config] = Array.isArray(args) ? args : [args];
    const res = await axiosInstance.get(url, { ...config });
    return res.data;
  } catch (error) {
    console.error('Failed to fetch:', error);
    throw error;
  }
};

export const endpoints = {
  auth: {
    login: '/login',
    register: '/register',
    me: '/me',
  },
};
// ----------------------------------------------------------------------

// export const endpoints = {
//   chat: '/api/chat',
//   kanban: '/api/kanban',
//   calendar: '/api/calendar',
//   auth: {
//     me: '/api/auth/me',
//     signIn: '/api/auth/sign-in',
//     signUp: '/api/auth/sign-up',
//   },
//   mail: {
//     list: '/api/mail/list',
//     details: '/api/mail/details',
//     labels: '/api/mail/labels',
//   },
//   post: {
//     list: '/api/post/list',
//     details: '/api/post/details',
//     latest: '/api/post/latest',
//     search: '/api/post/search',
//   },
//   product: {
//     list: '/api/product/list',
//     details: '/api/product/details',
//     search: '/api/product/search',
//   },
// };