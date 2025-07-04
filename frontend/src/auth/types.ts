// src\auth\types.ts
export type UserType = {
  id: string;
  email: string;
  isAdmin: number;
  role?: string; // 添加可选的role字段
} | null;

export type AuthState = {
  user: UserType;
  loading: boolean;
};

export type AuthContextValue = {
  user: UserType;
  loading: boolean;
  authenticated: boolean;
  unauthenticated: boolean;
  checkUserSession?: () => Promise<void>;
};

export type SignInRequest = {
  email: string;
  password: string;
};

export type SignUpParams = {
  email: string;
  password: string;
};

// 认证响应
export type JwtContextType = {
  token: string;
  user_id: string;
  is_admin: number;
  email: string; // 添加email字段
};