// src/components/table/data-table.tsx
import { useState, useCallback, useMemo, useEffect } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  TablePagination,
  Paper,
  Checkbox,
  Box,
  TextField,
  MenuItem,
  Select,
  InputAdornment,
  FormControl
} from '@mui/material';
import { Iconify } from 'src/components/iconify'; // 使用项目中已有的图标组件

// 定义列的接口
export interface Column {
  id: string;          // 列标识
  label: string;       // 列标题
  width?: string;      // 列宽度
  align?: 'left' | 'right' | 'center';  // 对齐方式
  sortable?: boolean;  // 是否可排序
  format?: (value: any) => React.ReactNode;  // 格式化函数
  sortType?: 'string' | 'number' | 'date' | 'numeric-string';  // 排序类型
}

// 搜索框组件属性接口
interface TableSearchProps {
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

// 表格搜索组件
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

// 组件属性接口
interface DataTableProps {
  columns: Column[];           // 列定义
  data: any[];                // 数据
  title?: string;             // 表格标题
  rowKey?: string;            // 行主键
  selectable?: boolean;       // 是否可选择
  onSelectionChange?: (selectedIds: string[]) => void;  // 选择变化回调
  searchable?: boolean;       // 是否启用搜索
  searchPosition?: 'left' | 'center' | 'right';  // 搜索框位置
  defaultSearchField?: string; // 默认搜索字段
  searchBoxPlacement?: 'inside' | 'above' | 'outside' | 'none'; // 搜索框放置位置
  onSearch?: (keyword: string, field: string) => void; // 搜索回调函数
}

type Order = 'asc' | 'desc';

// 主表格组件
export function DataTable({
  columns,
  data,
  title,
  rowKey = 'id',
  selectable = true,
  onSelectionChange,
  searchable = false,
  searchPosition = 'right',
  defaultSearchField,
  searchBoxPlacement = 'inside',
  onSearch: externalSearchHandler
}: DataTableProps) {
  // 排序状态
  const [orderBy, setOrderBy] = useState<string>('');
  const [order, setOrder] = useState<Order>('asc');
  
  // 选择状态
  const [selected, setSelected] = useState<string[]>([]);

  // 分页状态
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  
  // 搜索状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchField, setSearchField] = useState(defaultSearchField || '');

  // 处理搜索
  const handleSearch = useCallback((keyword: string, field: string) => {
    setSearchKeyword(keyword);
    setSearchField(field);
    setPage(0); // 重置到第一页
    
    // 如果有外部搜索处理函数，也调用它
    if (externalSearchHandler) {
      externalSearchHandler(keyword, field);
    }
  }, [externalSearchHandler]);

  // 处理排序
  const handleRequestSort = (property: string) => {
    const isAsc = orderBy === property && order === 'asc';
    setOrder(isAsc ? 'desc' : 'asc');
    setOrderBy(property);
  };

  // 处理全选
  const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      const filteredData = getFilteredData();
      const newSelected = filteredData.map(row => row[rowKey]);
      setSelected(newSelected);
      onSelectionChange?.(newSelected);
    } else {
      setSelected([]);
      onSelectionChange?.([]);
    }
  };

  // 处理单行选择
  const handleRowSelect = (id: string) => {
    const selectedIndex = selected.indexOf(id);
    let newSelected: string[] = [];

    if (selectedIndex === -1) {
      newSelected = newSelected.concat(selected, id);
    } else if (selectedIndex === 0) {
      newSelected = newSelected.concat(selected.slice(1));
    } else if (selectedIndex === selected.length - 1) {
      newSelected = newSelected.concat(selected.slice(0, -1));
    } else if (selectedIndex > 0) {
      newSelected = newSelected.concat(
        selected.slice(0, selectedIndex),
        selected.slice(selectedIndex + 1)
      );
    }

    setSelected(newSelected);
    onSelectionChange?.(newSelected);
  };

  // 处理页码变化
  const handleChangePage = (event: unknown, newPage: number) => {
    setPage(newPage);
  };

  // 处理每页行数变化
  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  // 过滤数据
  const getFilteredData = useCallback(() => {
    if (!searchKeyword || !searchField) return data;

    return data.filter(row => {
      const value = row[searchField];
      if (value === null || value === undefined) return false;
      
      return String(value).toLowerCase().includes(searchKeyword.toLowerCase());
    });
  }, [data, searchKeyword, searchField]);

  // 排序函数
  const sortData = (dataToSort: any[]): any[] => {
    if (!orderBy) return dataToSort;

    // 查找当前排序列的定义
    const currentColumn = columns.find(col => col.id === orderBy);
    const sortType = currentColumn?.sortType || 'string';

    return [...dataToSort].sort((a, b) => {
      // 处理空值
      if (a[orderBy] === null || a[orderBy] === undefined) return order === 'asc' ? -1 : 1;
      if (b[orderBy] === null || b[orderBy] === undefined) return order === 'asc' ? 1 : -1;

      // 根据不同类型进行排序
      if (sortType === 'numeric-string') {
        // 数字字符串排序（例如："1", "2", "10"）
        const numA = parseInt(a[orderBy], 10);
        const numB = parseInt(b[orderBy], 10);
        
        if (!isNaN(numA) && !isNaN(numB)) {
          // 如果两者都能转换为数字，按数字排序
          return order === 'asc' ? numA - numB : numB - numA;
        }
      } else if (sortType === 'number') {
        // 数字排序
        return order === 'asc' ? a[orderBy] - b[orderBy] : b[orderBy] - a[orderBy];
      } else if (sortType === 'date') {
        // 日期排序
        const dateA = new Date(a[orderBy]).getTime();
        const dateB = new Date(b[orderBy]).getTime();
        return order === 'asc' ? dateA - dateB : dateB - dateA;
      }

      // 默认字符串排序
      if (String(a[orderBy]).toLowerCase() < String(b[orderBy]).toLowerCase()) {
        return order === 'asc' ? -1 : 1;
      }
      if (String(a[orderBy]).toLowerCase() > String(b[orderBy]).toLowerCase()) {
        return order === 'asc' ? 1 : -1;
      }
      return 0;
    });
  };

  // 获取当前页的数据
  const getCurrentPageData = () => {
    const filteredData = getFilteredData();
    const sortedData = sortData(filteredData);
    return sortedData.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage);
  };

  // 计算过滤后的数据总数
  const filteredDataLength = useMemo(() => getFilteredData().length, [getFilteredData]);

  // 如果是outside或none模式
  if (searchable && (searchBoxPlacement === 'outside' || searchBoxPlacement === 'none')) {
    // 对于outside，返回搜索框组件
    if (searchBoxPlacement === 'outside') {
      return (
        <TableSearch
          columns={columns}
          onSearch={handleSearch}
          position={searchPosition}
          defaultField={defaultSearchField}
        />
      );
    }
    // 对于none，什么都不返回，但继续执行后面的表格渲染代码
  }

  return (
    <Paper sx={{ width: '100%', mb: 2 }}>
      {/* 搜索组件 - 当放在表格上方时 */}
      {searchable && searchBoxPlacement === 'above' && (
        <Box sx={{ p: 2, borderBottom: '1px solid rgba(0, 0, 0, 0.12)' }}>
          <TableSearch
            columns={columns}
            onSearch={handleSearch}
            position={searchPosition}
            defaultField={defaultSearchField}
          />
        </Box>
      )}
      
      <TableContainer 
        sx={{ 
          minHeight: '500px',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between'
        }}
      >
        {/* 搜索组件 - 当放在表格内部时 */}
        {searchable && searchBoxPlacement === 'inside' && (
          <Box sx={{ p: 2, borderBottom: '1px solid rgba(0, 0, 0, 0.12)' }}>
            <TableSearch
              columns={columns}
              onSearch={handleSearch}
              position={searchPosition}
              defaultField={defaultSearchField}
            />
          </Box>
        )}
        
        <Table sx={{ minWidth: 750 }}>
          <TableHead>
            <TableRow>
              {/* 复选框列 */}
              {selectable && (
                <TableCell padding="checkbox" sx={{ pl: 3 }}>
                  <Checkbox
                    indeterminate={selected.length > 0 && selected.length < filteredDataLength}
                    checked={filteredDataLength > 0 && selected.length === filteredDataLength}
                    onChange={handleSelectAllClick}
                  />
                </TableCell>
              )}
              
              {/* 数据列 */}
              {columns.map((column) => (
                <TableCell
                  key={column.id}
                  align={column.align || 'left'}
                  style={{ width: column.width }}
                  sx={{ 
                    height: 60,  // 修改行高
                    fontSize: '0.875rem',
                    fontWeight: 600,
                    borderBottom: '1px solid rgba(0, 0, 0, 0.12)',
                    whiteSpace: 'nowrap'
                  }}
                >
                  {column.sortable ? (
                    <TableSortLabel
                      active={orderBy === column.id}
                      direction={orderBy === column.id ? order : 'asc'}
                      onClick={() => handleRequestSort(column.id)}
                    >
                      {column.label}
                    </TableSortLabel>
                  ) : (
                    column.label
                  )}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>

          <TableBody>
            {getCurrentPageData().length > 0 ? (
              getCurrentPageData().map((row) => (
                <TableRow
                  key={row[rowKey]}
                  hover
                  selected={selected.indexOf(row[rowKey]) !== -1}
                >
                  {selectable && (
                    <TableCell padding="checkbox" sx={{ pl: 3 }}>
                      <Checkbox
                        checked={selected.indexOf(row[rowKey]) !== -1}
                        onClick={() => handleRowSelect(row[rowKey])}
                      />
                    </TableCell>
                  )}
                  
                  {columns.map((column) => (
                    <TableCell 
                      key={column.id} 
                      align={column.align}
                      sx={{ 
                        height: 60,  // 修改行高
                        fontSize: '0.875rem',
                        whiteSpace: 'nowrap'
                      }}
                    >
                      {column.format ? column.format(row[column.id]) : row[column.id]}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell 
                  colSpan={selectable ? columns.length + 1 : columns.length}
                  sx={{ 
                    textAlign: 'center',
                    py: 8,
                    fontSize: '0.875rem',
                    color: 'text.secondary'
                  }}
                >
                  暂无数据
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* 分页控件和选择计数器 */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          px: 2,
          py: 1,
          borderTop: '1px solid rgba(0, 0, 0, 0.12)'
        }}
      >
        {/* 选择计数器 */}
        {selectable && (
          <Box sx={{ minWidth: 100, whiteSpace: 'nowrap' }}>
            {selected.length > 0 ? `已选择${selected.length}项` : ''}
          </Box>
        )}

        {/* 分页控件 */}
        <TablePagination
          component="div"
          count={filteredDataLength}
          page={page}
          onPageChange={handleChangePage}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={handleChangeRowsPerPage}
          rowsPerPageOptions={[10, 25, 50, 100]}
          labelRowsPerPage="每页行数:"
          labelDisplayedRows={({ from, to, count }) => 
            `${from}-${to} 共 ${count}`
          }
        />
      </Box>
    </Paper>
  );
}