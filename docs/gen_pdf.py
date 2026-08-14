import os
import markdown
import subprocess

docs_dir = r'E:\2025-2026\2025-2026-2\wxx\WXX\docs'
md_file = os.path.join(docs_dir, '蔚小芯智能体学生操作手册.md')
html_file = os.path.join(docs_dir, '_manual.html')
pdf_file = os.path.join(docs_dir, '蔚小芯智能体学生操作手册v3.0.pdf')

with open(md_file, 'r', encoding='utf-8') as f:
    md_content = f.read()

html_body = markdown.markdown(
    md_content,
    extensions=['tables', 'fenced_code', 'toc'],
)

css = '''
@page {
  size: A4;
  margin: 18mm 16mm 18mm 16mm;
}
* { box-sizing: border-box; }
body {
  font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans SC", "WenQuanYi Micro Hei", sans-serif;
  font-size: 10.5pt;
  line-height: 1.65;
  color: #222;
  margin: 0;
}
h1 {
  font-size: 20pt;
  color: #1565C0;
  border-bottom: 2px solid #1565C0;
  padding-bottom: 6px;
  margin: 20px 0 14px;
  page-break-after: avoid;
}
h2 {
  font-size: 15pt;
  color: #1976D2;
  border-bottom: 1px solid #ccc;
  padding-bottom: 4px;
  margin: 16px 0 10px;
  page-break-after: avoid;
}
h3 {
  font-size: 12.5pt;
  color: #333;
  margin: 12px 0 8px;
  page-break-after: avoid;
}
h4 {
  font-size: 11pt;
  color: #444;
  margin: 10px 0 6px;
  page-break-after: avoid;
}
p { margin: 6px 0; }
table {
  border-collapse: collapse;
  width: 100%;
  margin: 10px 0;
  font-size: 9.5pt;
  page-break-inside: avoid;
}
th, td {
  border: 1px solid #ccc;
  padding: 5px 8px;
  text-align: left;
  vertical-align: top;
}
th {
  background: #eef3f8;
  font-weight: bold;
  color: #333;
}
tr { page-break-inside: avoid; }
tr:nth-child(even) { background: #fafafa; }
code {
  background: #f2f2f2;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 9.5pt;
  font-family: Consolas, monospace;
}
blockquote {
  border-left: 3px solid #1976D2;
  margin: 8px 0;
  padding: 6px 12px;
  background: #f6f9fc;
  color: #555;
  font-size: 9.5pt;
  page-break-inside: avoid;
}
div[align="center"] {
  text-align: center;
  margin: 10px 0;
  page-break-inside: avoid;
}
div[align="center"] img {
  max-width: 100%;
  height: auto;
  margin: 4px auto;
}
div[align="center"] table {
  margin-left: auto;
  margin-right: auto;
}
div[align="center"] table img {
  max-width: 100%;
  height: auto;
}
em {
  color: #666;
  font-size: 9pt;
  font-style: normal;
}
hr {
  border: none;
  border-top: 1px solid #ddd;
  margin: 14px 0;
}
strong { color: #1565C0; }
ul, ol { margin: 6px 0 6px 22px; padding: 0; }
li { margin: 3px 0; }
'''

full_html = f'''<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<style>{css}</style>
</head>
<body>
{html_body}
</body>
</html>'''

with open(html_file, 'w', encoding='utf-8') as f:
    f.write(full_html)

chrome = r'C:\Program Files\Google\Chrome\Application\chrome.exe'
if not os.path.exists(chrome):
    chrome = r'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'

file_url = 'file:///' + html_file.replace('\\', '/')
cmd = [
    chrome,
    '--headless=new',
    '--disable-gpu',
    '--no-pdf-header-footer',
    '--print-to-pdf=' + pdf_file,
    '--no-margins',
    file_url,
]
result = subprocess.run(cmd, capture_output=True, timeout=120)
print('Chrome exit code:', result.returncode)
if os.path.exists(pdf_file):
    size = os.path.getsize(pdf_file)
    print(f'PDF generated: {pdf_file}')
    print(f'Size: {size/1024:.0f} KB')
else:
    print('PDF generation failed')
    print(result.stderr[:2000])
