// src/components/table/table-search.tsx
import React, { useState, useEffect } from 'react';
import { Box, TextField, MenuItem, Select, InputAdornment, FormControl } from '@mui/material';
import { Iconify } from 'src/components/iconify'; // 使用项目中已有的图标组件
import { Column } from './data-table';

// 组件属性接口
export interface TableSearchProps {
  columns: Column[];                                    // 表格列定义
  onSearch: (keyword: string, field: string) => void;   // 搜索回调函数
  position?: 'left' | 'center' | 'right';               // 搜索框位置
  placeholder?: string;                                 // 输入框占位文本
  width?: string | number;                              // 搜索框宽度
  defaultField?: string;                                // 默认搜索字段
}

// 防抖函数
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

export function TableSearch({
  columns,
  onSearch,
  position = 'right',
  placeholder = '搜索...',
  width = 300,
  defaultField
}: TableSearchProps) {
  // 获取可搜索的列（排除带有格式化函数的列）
  const searchableColumns = columns.filter(col => !col.format);
  
  // 状态管理
  const [keyword, setKeyword] = useState<string>('');
  const [field, setField] = useState<string>(defaultField || (searchableColumns.length > 0 ? searchableColumns[0].id : ''));
  
  // 使用防抖处理搜索关键词
  const debouncedKeyword = useDebounce(keyword, 300);
  
  // 当搜索关键词或字段变化时触发搜索
  useEffect(() => {
    onSearch(debouncedKeyword, field);
  }, [debouncedKeyword, field, onSearch]);
  
  // 处理输入变化
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setKeyword(e.target.value);
  };
  
  // 处理字段选择变化
  const handleFieldChange = (e: React.ChangeEvent<{ value: unknown }>) => {
    setField(e.target.value as string);
  };
  
  // 根据位置设置对齐方式
  const getPositionStyle = () => {
    switch (position) {
      case 'left':
        return { justifyContent: 'flex-start' };
      case 'center':
        return { justifyContent: 'center' };
      case 'right':
      default:
        return { justifyContent: 'flex-end' };
    }
  };
  
  return (
    <Box 
      sx={{ 
        display: 'flex', 
        mb: 2,
        ...getPositionStyle()
      }}
    >
      <Box 
        sx={{ 
          display: 'flex',
          width: typeof width === 'number' ? `${width}px` : width,
          border: '1px solid rgba(0, 0, 0, 0.12)',
          borderRadius: 1,
          overflow: 'hidden'
        }}
      >
        <TextField
          variant="outlined"
          size="small"
          placeholder={placeholder}
          value={keyword}
          onChange={handleInputChange}
          sx={{ 
            flex: 1,
            '& .MuiOutlinedInput-root': {
              '& fieldset': {
                border: 'none'
              }
            }
          }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <Iconify icon="eva:search-fill" width={20} height={20} />
              </InputAdornment>
            ),
          }}
        />
        
        <FormControl 
          variant="outlined" 
          size="small"
          sx={{ 
            minWidth: 120,
            '& .MuiOutlinedInput-root': {
              '& fieldset': {
                border: 'none',
                borderLeft: '1px solid rgba(0, 0, 0, 0.12)'
              }
            }
          }}
        >
          <Select
            value={field}
            onChange={handleFieldChange as any}
            displayEmpty
          >
            {searchableColumns.map((column) => (
              <MenuItem key={column.id} value={column.id}>
                {column.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>
    </Box>
  );
}
