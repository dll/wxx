"""Generate adaptive icon foreground PNGs for Android launcher icons."""
from PIL import Image, ImageDraw, ImageFont
import os

BASE = r'E:\2025-2026\2025-2026-2\wxx\WXX\frontend\android\app\src\main\res'

# Foreground sizes per density (adaptive icon: full 108x108 dp area)
# mdpi = 1x, hdpi = 1.5x, xhdpi = 2x, xxhdpi = 3x, xxxhdpi = 4x
SIZES = {
    'mipmap-mdpi': 108,
    'mipmap-hdpi': 162,
    'mipmap-xhdpi': 216,
    'mipmap-xxhdpi': 324,
    'mipmap-xxxhdpi': 432,
}

FONT_PATH = r'C:\Windows\Fonts\msyh.ttc'

def draw_foreground(size):
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Safe zone: 72/108 = 66.67% of full size, centered
    safe_ratio = 0.667
    safe_size = int(size * safe_ratio)
    ox = (size - safe_size) // 2
    oy = (size - safe_size) // 2

    # Draw "芯" character
    char_size = int(safe_size * 0.62)
    try:
        font = ImageFont.truetype(FONT_PATH, char_size)
    except:
        font = ImageFont.load_default()

    char = '芯'
    bbox = draw.textbbox((0, 0), char, font=font)
    cw = bbox[2] - bbox[0]
    ch = bbox[3] - bbox[1]
    cx = ox + (safe_size - cw) // 2 - bbox[0]
    cy = oy + (safe_size - ch) // 2 - bbox[1] - int(size * 0.03)
    draw.text((cx, cy), char, fill='white', font=font)

    # Sparkle star above character
    sparkle_size = int(size * 0.12)
    sparkle_center = (size // 2, oy + int(safe_size * 0.18))
    _draw_sparkle(draw, sparkle_center, sparkle_size)

    return img

def _draw_sparkle(draw, center, size):
    cx, cy = center
    hs = size / 2
    points = [
        (cx, cy - hs),      # top
        (cx + hs * 0.25, cy - hs * 0.25),
        (cx + hs, cy),      # right
        (cx + hs * 0.25, cy + hs * 0.25),
        (cx, cy + hs),      # bottom
        (cx - hs * 0.25, cy + hs * 0.25),
        (cx - hs, cy),      # left
        (cx - hs * 0.25, cy - hs * 0.25),
    ]
    draw.polygon(points, fill='white')

def main():
    icon_dir = os.path.dirname(__file__)
    for folder, size in SIZES.items():
        path = os.path.join(BASE, folder, 'ic_launcher_foreground.png')
        img = draw_foreground(size)
        img.save(path, 'PNG')
        print(f'Created {folder}/ic_launcher_foreground.png ({size}x{size})')

if __name__ == '__main__':
    main()
