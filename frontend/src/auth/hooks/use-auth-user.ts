// src/auth/hooks/use-auth-user.ts
import { useAuthContext } from './use-auth-context';
import { _mock } from 'src/_mock';

// ----------------------------------------------------------------------

export function useAuthUser() {
  const { user } = useAuthContext();
  
  return {
    user: user ? {
      id: user.id,
      displayName: user.email?.split('@')[0] || 'User', // 使用邮箱前缀作为显示名称
      email: user.email,
      photoURL: _mock.image.avatar(1), // 使用模拟头像而不是 null
      isAdmin: user.isAdmin,
      role: user.isAdmin === 1 ? 'admin' : 'user'
    } : {
      // 提供默认值以避免类型错误
      id: '',
      displayName: 'Guest',
      email: '',
      photoURL: _mock.image.avatar(1),
      isAdmin: 0,
      role: 'guest'
    }
  };
}