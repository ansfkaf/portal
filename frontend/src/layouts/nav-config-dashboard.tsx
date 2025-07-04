// src/layouts/nav-config-dashboard.tsx - 更新导航配置
import type { NavSectionProps } from 'src/components/nav-section';
import { paths } from 'src/routes/paths';
import { CONFIG } from 'src/global-config';
import { Label } from 'src/components/label';
import { SvgColor } from 'src/components/svg-color';
import { routeConfig } from 'src/routes/sections/dashboard';

// ----------------------------------------------------------------------

const icon = (name: string) => (
  <SvgColor src={`${CONFIG.assetsDir}/assets/icons/navbar/${name}.svg`} />
);

const ICONS = {
  job: icon('ic-job'),
  blog: icon('ic-blog'),
  chat: icon('ic-chat'),
  mail: icon('ic-mail'),
  user: icon('ic-user'),
  file: icon('ic-file'),
  lock: icon('ic-lock'),
  tour: icon('ic-tour'),
  order: icon('ic-order'),
  label: icon('ic-label'),
  blank: icon('ic-blank'),
  kanban: icon('ic-kanban'),
  folder: icon('ic-folder'),
  course: icon('ic-course'),
  banking: icon('ic-banking'),
  booking: icon('ic-booking'),
  invoice: icon('ic-invoice'),
  product: icon('ic-product'),
  calendar: icon('ic-calendar'),
  disabled: icon('ic-disabled'),
  external: icon('ic-external'),
  menuItem: icon('ic-menu-item'),
  ecommerce: icon('ic-ecommerce'),
  analytics: icon('ic-analytics'),
  dashboard: icon('ic-dashboard'),
  parameter: icon('ic-parameter'),
  setting: icon('ic-setting'),
};

// 导航项配置
const navItems = [
  {
    title: '首页',
    path: paths.dashboard.root,
    icon: ICONS.dashboard,
    requiredRole: 'all',
  },
  {
    title: '账号管理',
    path: paths.dashboard.account,
    icon: ICONS.user,
    requiredRole: 'admin',
  },
  { 
    title: '实例管理', 
    path: paths.dashboard.instance, 
    icon: ICONS.parameter,
    requiredRole: 'admin',
  },
  {
    title: '在线实例',
    path: paths.dashboard.pool,
    icon: ICONS.analytics,
    requiredRole: 'all',
  },
  { 
    title: '账户导入', 
    path: paths.dashboard.import, 
    icon: ICONS.file,
    requiredRole: 'admin',
  },
  {
    title: '监控管理',
    path: paths.dashboard.monitor,
    icon: ICONS.banking,
    requiredRole: 'all',
  },
  {
    title: '开机设置',
    path: paths.dashboard.setting,
    icon: ICONS.parameter,
    requiredRole: 'all',
  },
  {
    title: '用户管理',
    path: paths.dashboard.user,
    icon: ICONS.user,
    requiredRole: 'admin',
  },
  {
    title: '数据库管理', // 新增：数据库管理导航项
    path: paths.dashboard.database,
    icon: ICONS.folder, // 使用文件夹图标或其他适合的图标
    requiredRole: 'admin',
  },
  // 修改队列管理菜单配置
  {
    title: '队列管理',
    path: paths.dashboard.queue.root, // 使用根路径
    icon: ICONS.kanban,
    requiredRole: 'admin',
    children: [
      { 
        title: '账号池队列', 
        path: paths.dashboard.queue.accountPool,
        requiredRole: 'admin',
      },
      { 
        title: '补机队列', // 新增：补机队列导航项
        path: paths.dashboard.queue.makeupQueue, 
        requiredRole: 'admin',
      },
      { 
        title: '补机历史', 
        path: paths.dashboard.queue.makeupHistory,
        requiredRole: 'admin',
      },
    ],
  }
];

// 根据用户角色获取导航数据
export const getNavData = (isAdmin: boolean): NavSectionProps['data'] => {
  const userRole = isAdmin ? 'admin' : 'user';
  
  // 过滤导航项，只显示用户有权限访问的菜单
  const filteredItems = navItems.filter(item => {
    // 检查主菜单项权限
    const hasMainPermission = item.requiredRole === 'all' || item.requiredRole === userRole;
    
    // 如果有子菜单，需要过滤子菜单
    if (hasMainPermission && item.children) {
      item.children = item.children.filter(
        child => child.requiredRole === 'all' || child.requiredRole === userRole
      );
      
      // 如果过滤后没有子菜单，则不显示主菜单
      if (item.children.length === 0) {
        return false;
      }
    }
    
    return hasMainPermission;
  });
  
  return [
    {
      subheader: '导航菜单',
      items: filteredItems,
    },
  ];
};