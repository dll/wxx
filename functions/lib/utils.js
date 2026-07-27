// 工具函数

export function stringToHex(str) {
  return Array.from(str)
    .map(c => c.charCodeAt(0).toString(16).padStart(2, '0'))
    .join('');
}

export function hexToString(hex) {
  const bytes = [];
  for (let i = 0; i < hex.length; i += 2) {
    bytes.push(parseInt(hex.substr(i, 2), 16));
  }
  return String.fromCharCode(...bytes);
}

export function generateId() {
  return Date.now().toString(36) + Math.random().toString(36).substr(2, 9);
}

export function now() {
  return new Date().toISOString();
}

export function paginate(list, page, pageSize) {
  const start = (page - 1) * pageSize;
  const end = start + pageSize;
  return {
    items: list.slice(start, end),
    total: list.length,
    page,
    page_size: pageSize,
  };
}
