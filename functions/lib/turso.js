// Turso 数据库客户端（通过 HTTP API 访问）
// 适用于 Cloudflare Pages Functions / Workers 环境

// Hrana v2 协议要求 integer 的 value 必须是 JSON 字符串（避免 i64 精度丢失），
// 传原始数字会被服务端拒绝：expected a borrowed string。
function encodeArg(a) {
  if (a === null || a === undefined) return { type: 'null', value: null };
  const t = typeof a;
  if (t === 'bigint') return { type: 'integer', value: a.toString() };
  if (t === 'number') {
    return Number.isInteger(a)
      ? { type: 'integer', value: String(a) }
      : { type: 'float', value: a };
  }
  if (t === 'boolean') return { type: 'integer', value: a ? '1' : '0' };
  return { type: 'text', value: String(a) };
}

export class TursoClient {
  constructor(dbUrl, authToken) {
    // 将 libsql:// 转换为 https://
    this.baseUrl = dbUrl.replace('libsql://', 'https://');
    this.authToken = authToken;
  }

  async execute(sql, args = []) {
    const url = `${this.baseUrl}/v2/pipeline`;
    const body = {
      requests: [
        {
          type: 'execute',
          stmt: {
            sql,
            args: args.map(encodeArg),
          },
        },
      ],
    };

    const resp = await fetch(url, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`Turso API error ${resp.status}: ${text}`);
    }

    const data = await resp.json();
    if (data.results && data.results[0] && data.results[0].response) {
      const result = data.results[0].response.result;
      return this.parseResult(result);
    }
    throw new Error('Unexpected Turso response format');
  }

  async query(sql, args = []) {
    return this.execute(sql, args);
  }

  parseResult(result) {
    if (!result || !result.rows || !result.cols) {
      return { rows: [], columns: [] };
    }

    const columns = result.cols.map(c => c.name);
    const rows = result.rows.map(row => {
      const obj = {};
      row.forEach((cell, i) => {
        const key = columns[i];
        if (cell === null || cell === undefined || cell.type === 'null') {
          obj[key] = null;
        } else if (cell.type === 'integer') {
          // Hrana v2 返回整数列时 value 是字符串（避免 i64 精度丢失）；
          // 统一转为 JS number，调用方无需再手动 parseInt。
          obj[key] = parseInt(cell.value, 10);
        } else if (cell.type === 'float') {
          obj[key] = parseFloat(cell.value);
        } else if (typeof cell.value !== 'undefined') {
          obj[key] = cell.value;
        } else {
          obj[key] = cell;
        }
      });
      return obj;
    });

    return { rows, columns };
  }

  async getOne(sql, args = []) {
    const result = await this.query(sql, args);
    return result.rows.length > 0 ? result.rows[0] : null;
  }

  async getAll(sql, args = []) {
    const result = await this.query(sql, args);
    return result.rows;
  }
}
