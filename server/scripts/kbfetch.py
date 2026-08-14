#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
kbfetch.py — 学院/学校官网知识抓取入库生成器

从指定栏目（webplus 建站系统，如 csci.chzu.edu.cn）抓取真实文章，
清洗正文后生成 kb_resources 的种子 SQL 迁移（INSERT OR IGNORE），
由 migrate 命令统一执行。

原则（准确性第一）：
  - 正文取自官网原文，不编造、不改写要点
  - resource_id 用文章页码号保证唯一、可重复执行（INSERT OR IGNORE）
  - source_link 指向真实文章 URL，source_version 标发布时间
  - 生成后建议人工抽查若干条再执行 migrate

依赖：requests、beautifulsoup4（如缺： pip install requests beautifulsoup4）
用法：  python server/scripts/kbfetch.py
"""
import sys, io, os, re, time, json, argparse
sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')
import requests
from bs4 import BeautifulSoup

USER_AGENT = ('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 '
              '(KHTML, like Gecko) Chrome/120.0 Safari/537.36')

# 栏目映射：名称 / 列表页 URL / 入库 resource_type / 基础标签 / 抓取页数
COLUMNS = [
    dict(name="资助管理", url="https://csci.chzu.edu.cn/zzgl/list.htm",
         rtype="FAQ", tags=["资助", "助学", "困难认定"], pages=2),
    dict(name="学工动态", url="https://csci.chzu.edu.cn/xgdt/list.htm",
         rtype="Activity", tags=["学工", "学生工作"], pages=2),
    dict(name="就业指导", url="https://csci.chzu.edu.cn/jyzd/list.htm",
         rtype="FAQ", tags=["就业", "求职"], pages=2),
    dict(name="心理健康", url="https://csci.chzu.edu.cn/xljk/list.htm",
         rtype="FAQ", tags=["心理", "健康"], pages=1),
    dict(name="学科竞赛", url="https://csci.chzu.edu.cn/xkjs/list.htm",
         rtype="Activity", tags=["竞赛", "学科竞赛"], pages=1),
    dict(name="团学工作", url="https://csci.chzu.edu.cn/tzzdt/list.htm",
         rtype="Activity", tags=["团学", "活动"], pages=1),
    # 第二批：高年级/学业/就业主题
    dict(name="考研事迹", url="https://csci.chzu.edu.cn/kysj/list.htm",
         rtype="FAQ", tags=["考研", "上岸", "经验"], pages=1),
    dict(name="大创项目", url="https://csci.chzu.edu.cn/3582/list.htm",
         rtype="FAQ", tags=["大创", "科研", "项目"], pages=1),
    dict(name="毕设工作", url="https://csci.chzu.edu.cn/bysj/list.htm",
         rtype="Process", tags=["毕设", "毕业设计"], pages=1),
    dict(name="学风建设", url="https://csci.chzu.edu.cn/sqyr/list.htm",
         rtype="Activity", tags=["学风", "三全育人"], pages=1),
    dict(name="研究生通知", url="https://csci.chzu.edu.cn/yjstz/list.htm",
         rtype="FAQ", tags=["研究生", "通知"], pages=1),
    dict(name="学科建设", url="https://csci.chzu.edu.cn/xkjj/list.htm",
         rtype="FAQ", tags=["学科", "专业"], pages=1),
]

SESSION = requests.Session()
SESSION.headers.update({"User-Agent": USER_AGENT,
                        "Accept-Language": "zh-CN,zh;q=0.9"})


def get(url, tries=3):
    for _ in range(tries):
        try:
            r = SESSION.get(url, timeout=20)
            if r.status_code == 200:
                return r
        except Exception:
            time.sleep(1.5)
    return None


def list_articles(col, limit=0):
    """抓取栏目列表（含分页），返回 [(title, url)] 去重。"""
    items, seen, url = [], set(), col["url"]
    for _ in range(max(1, col["pages"])):
        r = get(url)
        if not r:
            break
        r.encoding = r.apparent_encoding or r.encoding or "utf-8"
        soup = BeautifulSoup(r.text, "html.parser")
        for a in soup.find_all("a", href=True):
            href = a["href"].strip()
            txt = re.sub(r"\s+", " ", a.get_text(" ", strip=True))
            if ("/info/" in href or re.search(r"/\d{4}/[^/]+/page\.htm", href)) \
               and len(txt) >= 4 and not txt.startswith("更多"):
                full = href if href.startswith("http") else \
                    (col["url"] if href.startswith("/") else "/" + href)
                # 解析相对链接
                from urllib.parse import urljoin
                full = urljoin(col["url"], href)
                if full not in seen:
                    seen.add(full)
                    items.append((txt, full))
                    if limit and len(items) >= limit:
                        return items
        # 下一页
        nxt = None
        for a in soup.find_all("a", href=True):
            if "下一页" in a.get_text(strip=True) and "list" in a["href"]:
                nxt = a["href"]
                break
        if not nxt:
            break
        url = urljoin(col["url"], nxt)
    return items


ARTICLE_ID = re.compile(r"/a(\d{5,})/page\.htm")


def fetch_article(article_url):
    """返回 (content, date)。正文区优先取 wp_articlecontent，兼容 v_news_content。"""
    r = get(article_url)
    if not r:
        return "", ""
    r.encoding = r.apparent_encoding or r.encoding or "utf-8"
    soup = BeautifulSoup(r.text, "html.parser")
    node = (soup.find("div", class_=re.compile(r"wp_articlecontent"))
            or soup.find("div", class_=re.compile(r"v_news_content"))
            or soup.find("div", class_=re.compile(r"articlecontent")))
    if node:
        content = re.sub(r"\n\s*\n+", "\n", node.get_text("\n", strip=True)).strip()
    else:
        for cls in ("nav", "footer", "header", "sidebar", "menu"):
            for el in soup.find_all(class_=re.compile(cls, re.I)):
                el.decompose()
        content = re.sub(r"\n\s*\n+", "\n", soup.get_text("\n", strip=True)).strip()
    # 去掉首尾噪声行（重复标题、作者/发布信息、浏览量）
    lines = [ln.strip() for ln in content.split("\n") if ln.strip()]
    # 去掉结尾的 版权所有/通讯员/初审/终审/电话 等页脚行
    keep = []
    for ln in lines:
        if re.match(r"^(版权所有|Copyright|邮编|联系|E-?Mail|初审|终审|通讯员|拟稿|编辑|审核|发布人|一审|二审|Final|TEL|电话)", ln, re.I):
            continue
        keep.append(ln)
    content = "\n".join(keep)
    # 发布日期
    date = ""
    m = re.search(r"发布时间[:：]\s*(\d{4}-\d{2}-\d{2})", soup.get_text())
    if m:
        date = m.group(1)
    if not date:
        m2 = soup.find("meta", attrs={"name": "publishdate"})
        if m2 and m2.get("content"):
            date = m2["content"].strip()
    return content, date


def esc(s):
    return s.replace("'", "''").replace("\\", "\\\\")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=None, help="输出 SQL 路径（默认 server/migrations/082_fetched_kb.sql）")
    ap.add_argument("--limit", type=int, default=0, help="每栏目最多条数（0=不限）")
    ap.add_argument("--cols", default="", help="只抓指定栏目（逗号分隔，默认全部）")
    ap.add_argument("--dry", action="store_true", help="只打印不写文件")
    args = ap.parse_args()

    base = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..")
    out = args.out or os.path.join(base, "server", "migrations", "083_fetched_kb2.sql")
    want = set(x.strip() for x in args.cols.split(",") if x.strip())

    rows = []
    seen_id = set()
    for col in COLUMNS:
        if want and col["name"] not in want:
            continue
        print(f"== 抓取栏目【{col['name']}】 {col['url']}")
        items = list_articles(col, args.limit)
        print(f"   列表 {len(items)} 篇，逐篇抓正文…")
        for title, u in items:
            m = ARTICLE_ID.search(u)
            rid = f"fetched-{col['name']}-{m.group(1)}" if m else \
                f"fetched-{col['name']}-{int(time.time()*1000)}"
            if rid in seen_id:
                continue
            seen_id.add(rid)
            content, date = fetch_article(u)
            if len(content) < 30:
                print(f"   - 跳过（正文过短）: {title[:28]}")
                continue
            runes = list(content)
            summary = content if len(runes) <= 80 else content[:80] + "…"
            rows.append(dict(rid=rid, rtype=col["rtype"], title=title,
                             summary=summary, content=content, link=u,
                             date=date, tags=col["tags"]))
            time.sleep(0.6)
    print(f"共抓取 {len(rows)} 条真实内容")

    if args.dry:
        for r in rows[:5]:
            print("\n###", r["title"])
            print(r["content"][:120])
        return

    lines = [
        "-- 082_fetched_kb.sql — 由 server/scripts/kbfetch.py 从学院官网抓取生成",
        "-- 来源：滁州学院计算机科学与工程学院官网(https://csci.chzu.edu.cn) 真实栏目",
        "-- 原则：正文取自官网原文，不编造；source_link 指向真实文章 URL；可重复执行(INSERT OR IGNORE)",
        "",
        "INSERT OR IGNORE INTO kb_resources",
        "(resource_id, resource_type, owner_scope, owner_id, role_scope, version, status, title, summary, content, source_link, source_version, tags, updated_by)",
        "VALUES",
    ]
    parts = [
        "('{rid}','{rtype}','college','cs','[\"student\",\"counselor\",\"student_union\",\"college_admin\"]','v1.0','published',\n"
        " '{title}',\n"
        " '{summary}',\n"
        " '{content}',\n"
        " '{link}',\n"
        " '{date}',\n"
        " '{tags}','kbfetch')".format(
            rid=esc(r["rid"]), rtype=r["rtype"], title=esc(r["title"]),
            summary=esc(r["summary"]), content=esc(r["content"]),
            link=esc(r["link"]), date=esc(r["date"]),
            tags=esc(json.dumps(r["tags"], ensure_ascii=False)))
        for r in sorted(rows, key=lambda x: x["rid"])
    ]
    lines.append(",\n".join(parts))
    lines.append(";")
    sql = "\n".join(lines)

    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, "w", encoding="utf-8", newline="\n") as f:
        f.write(sql)
    print(f"已写出 {out}（{len(rows)} 条）")


if __name__ == "__main__":
    main()
