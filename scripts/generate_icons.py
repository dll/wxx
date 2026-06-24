"""Generate 蔚小芯 icon PNGs for Android launcher and Web manifest."""

import math
import os
from PIL import Image, ImageDraw, ImageFont

FONT_PATH = r"C:\Windows\Fonts\msyh.ttc"
CHAR = "芯"

OUTPUTS = [
    # (path, size, is_maskable)
    (r"..\frontend\android\app\src\main\res\mipmap-mdpi\ic_launcher.png", 48, False),
    (r"..\frontend\android\app\src\main\res\mipmap-hdpi\ic_launcher.png", 72, False),
    (r"..\frontend\android\app\src\main\res\mipmap-xhdpi\ic_launcher.png", 96, False),
    (r"..\frontend\android\app\src\main\res\mipmap-xxhdpi\ic_launcher.png", 144, False),
    (r"..\frontend\android\app\src\main\res\mipmap-xxxhdpi\ic_launcher.png", 192, False),
    (r"..\frontend\web\favicon.png", 48, False),
    (r"..\frontend\web\icons\Icon-192.png", 192, False),
    (r"..\frontend\web\icons\Icon-512.png", 512, False),
    (r"..\frontend\web\icons\Icon-maskable-192.png", 192, True),
    (r"..\frontend\web\icons\Icon-maskable-512.png", 512, True),
]

COLOR_A = (26, 35, 126)   # #1A237E deep navy
COLOR_B = (66, 165, 245)  # #42A5F5 light azure

def lerp_color(c1, c2, t):
    return tuple(int(a + (b - a) * t) for a, b in zip(c1, c2))

def make_gradient(size):
    img = Image.new("RGBA", (size, size))
    cx, cy = size / 2, size / 2
    max_r = math.sqrt(2) * size / 2
    for y in range(size):
        for x in range(size):
            dx, dy = x - cx, y - cy
            r = math.sqrt(dx * dx + dy * dy) / max_r
            # radial gradient from top-left weighted
            tx = x / (size - 1) if size > 1 else 0
            ty = y / (size - 1) if size > 1 else 0
            t = (tx + ty) / 2
            c = lerp_color(COLOR_A, COLOR_B, t)
            img.putpixel((x, y), c + (255,))
    return img

def make_sparkle(draw, cx, cy, r, color="white"):
    """Draw a 4-pointed sparkle star."""
    points = []
    for i in range(8):
        angle = math.pi / 4 * i
        radius = r if i % 2 == 0 else r * 0.35
        points.append((cx + math.cos(angle) * radius, cy + math.sin(angle) * radius))
    draw.polygon(points, fill=color)

def render_icon(path, size, is_maskable):
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(canvas)

    # Background gradient
    bg = make_gradient(size)
    canvas.paste(bg, (0, 0), bg)

    # Clip to rounded square (adaptive icon style)
    mask = Image.new("L", (size, size), 0)
    mask_draw = ImageDraw.Draw(mask)
    r = size / 4
    mask_draw.rounded_rectangle([(0, 0), (size - 1, size - 1)], radius=r, fill=255)
    canvas.putalpha(mask)

    # Safe zone center
    safe_margin = size * 0.1 if is_maskable else 0
    safe_size = size - 2 * safe_margin
    center_x = size / 2

    # Character "芯" — takes about 60% of safe zone height
    char_h = int(safe_size * 0.60) if is_maskable else int(size * 0.60)
    try:
        font = ImageFont.truetype(FONT_PATH, char_h)
    except Exception:
        font = ImageFont.load_default()
    # Get bounding box
    bbox = draw.textbbox((0, 0), CHAR, font=font)
    tw, th = bbox[2] - bbox[0], bbox[3] - bbox[1]
    char_x = center_x - tw / 2
    char_y = (size - th) / 2 - 2  # slight upward offset for sparkle
    draw.text((char_x, char_y), CHAR, fill="white", font=font)

    # Sparkle star above character
    sparkle_r = size * 0.08
    sparkle_y = char_y - sparkle_r * 2.5
    make_sparkle(draw, center_x, sparkle_y, sparkle_r)

    # Subtle white border ring (1px at 48 scale, proportional)
    border_w = max(1, round(size / 48))
    offset = border_w / 2
    draw.rounded_rectangle(
        [offset, offset, size - 1 - offset, size - 1 - offset],
        radius=r,
        outline=(255, 255, 255, 160),
        width=border_w,
    )

    # Save
    dir_name = os.path.dirname(path)
    if dir_name:
        os.makedirs(dir_name, exist_ok=True)
    canvas.save(path, "PNG")
    print(f"  OK  {path}  ({size}x{size})")

def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    print("Generating 蔚小芯 icons...")
    for rel_path, size, maskable in OUTPUTS:
        full_path = os.path.normpath(os.path.join(script_dir, rel_path))
        render_icon(full_path, size, maskable)
    print("Done - all 10 icons generated.")

if __name__ == "__main__":
    main()
