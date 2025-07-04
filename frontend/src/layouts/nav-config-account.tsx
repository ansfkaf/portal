// src/layouts/nav-config-account.tsx
import { Iconify } from 'src/components/iconify';

import type { AccountDrawerProps } from './components/account-drawer';

// ----------------------------------------------------------------------

export const _account: AccountDrawerProps['data'] = [
  { label: '首页', href: '/', icon: <Iconify icon="solar:home-angle-bold-duotone" /> },
  // 只保留首页导航项，其他全部删除
];