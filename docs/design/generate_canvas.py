#!/usr/bin/env python3
"""Cabinet of Equivalents — a visual plate for the design philosophy.
Renders a museum-grade taxonomy sheet as PNG + PDF using Pillow.
Limited palette: bone paper, iron ink, oxidised green, burnt amber.
"""
import math, random
import numpy as np
from PIL import Image, ImageDraw, ImageFont

FONT_DIR = "/Users/chenyang/.config/magpie/library/skills/canvas-design/canvas-fonts"
OUT = "/Users/chenyang/Documents/github-repos/SwiftInstall/docs/design"

# ---- palette ----
PAPER = (243, 238, 228)
INK   = (26, 25, 22)
MUTE  = (138, 132, 120)
LINE  = (204, 196, 180)
GREEN = (48, 104, 77)
AMBER = (176, 122, 42)

W, H = 2400, 3400
M = 96
PAD = 70
GX0 = M + PAD
GX1 = W - M - PAD

def f(name, size):
    return ImageFont.truetype(f"{FONT_DIR}/{name}", size)

TITLE_IT = f("InstrumentSerif-Italic.ttf", 96)
TITLE    = f("Gloock-Regular.ttf", 206)
MONO     = f("DMMono-Regular.ttf", 24)
MONO_S   = f("DMMono-Regular.ttf", 17)
MONO_XS  = f("DMMono-Regular.ttf", 18)
LEG_T    = f("DMMono-Regular.ttf", 22)
LEG_S    = f("DMMono-Regular.ttf", 16)

# ---- canvas with paper grain ----
img = Image.new("RGB", (W, H), PAPER)
arr = np.array(img).astype(np.float32)
rng = np.random.default_rng(7)
noise = rng.normal(0, 3.2, (H, W, 1))
arr = np.clip(arr + noise, 0, 255).astype(np.uint8)
img = Image.fromarray(arr)
draw = ImageDraw.Draw(img)

# very subtle vignette for print depth
yy, xx = np.mgrid[0:H, 0:W]
cx, cy = W / 2, H / 2
d = np.sqrt(((xx - cx) / (W * 0.72)) ** 2 + ((yy - cy) / (H * 0.72)) ** 2)
vig = np.clip(d, 0, 1)[..., None]
arr = np.array(img).astype(np.float32)
arr = np.clip(arr * (1 - 0.05 * vig), 0, 255).astype(np.uint8)
img = Image.fromarray(arr)
draw = ImageDraw.Draw(img)

def tracked(x, y, text, font, fill, tr=2.0, anchor="l"):
    widths = [draw.textlength(c, font=font) for c in text]
    total = sum(widths) + tr * (len(text) - 1)
    if anchor == "c":
        x -= total / 2
    elif anchor == "r":
        x -= total
    for c, w in zip(text, widths):
        draw.text((x, y), c, font=font, fill=fill, anchor="lm" if False else "ls")
        x += w + tr
    return total

def tracked_center(x, y, text, font, fill, tr=2.0):
    # center anchored on left-middle baseline coordinate y
    tracked(x, y, text, font, fill, tr, anchor="c")

# ---- frame ----
draw.rectangle([M, M, W - M, H - M], outline=INK, width=2)
draw.rectangle([M + 13, M + 13, W - M - 13, H - M - 13], outline=LINE, width=1)

# corner registration marks
def crop_marks():
    L = 22
    off = M + 34
    for (px, py, sx, sy) in [(off, off, 1, 1), (W - off, off, -1, 1),
                             (off, H - off, 1, -1), (W - off, H - off, -1, -1)]:
        draw.line([(px, py), (px + sx * L, py)], fill=MUTE, width=1)
        draw.line([(px, py), (px, py + sy * L)], fill=MUTE, width=1)
crop_marks()

# ---- header ----
hy = M + 52
tracked(GX0, hy, "CGARTLAB  ·  INSTRUMENT LEDGER", MONO_XS, INK, tr=3.0)
txt = "PLATE VII  ·  MMXXVI"
w = sum(draw.textlength(c, font=MONO_XS) for c in txt) + 3.0 * (len(txt) - 1)
tracked(GX1 - w, hy, txt, MONO_XS, INK, tr=3.0)

# ---- title ----
y = M + 148
a1, d1 = TITLE_IT.getmetrics()
draw.text((GX0, y), "The Cabinet of", font=TITLE_IT, fill=MUTE, anchor="la")
y += (a1 + d1) + 18
tw = draw.textlength("Equivalents", font=TITLE)
if tw > (GX1 - GX0):
    new_size = int(206 * (GX1 - GX0) / tw)
    globals()["TITLE"] = f("Gloock-Regular.ttf", new_size)
a2, d2 = TITLE.getmetrics()
draw.text((GX0 - 6, y), "Equivalents", font=TITLE, fill=INK, anchor="la")
y += (a2 + d2) + 40

# amber short rule
draw.line([(GX0, y), (GX0 + 132, y)], fill=AMBER, width=2)
y += 32
tracked(GX0, y, "OBSERVED  ·  EVALUATED  ·  QUIETLY  REPLACED", MONO, GREEN, tr=2.4)
y += 46

# full divider
draw.line([(GX0, y), (GX1, y)], fill=LINE, width=1)
y += 58

# ---- specimen grid ----
grid_top = y
grid_bottom = H - M - 372
cols, rows = 5, 7
cw = (GX1 - GX0) / cols
ch = (grid_bottom - grid_top) / rows

def draw_shape(d, cx, cy, r, kind, filled, color, width=2):
    if kind == "circle":
        bbox = [cx - r, cy - r, cx + r, cy + r]
        if filled: d.ellipse(bbox, fill=color)
        else: d.ellipse(bbox, outline=color, width=width)
    elif kind == "square":
        bbox = [cx - r, cy - r, cx + r, cy + r]
        if filled: d.rectangle(bbox, fill=color)
        else: d.rectangle(bbox, outline=color, width=width)
    elif kind == "diamond":
        pts = [(cx, cy - r), (cx + r, cy), (cx, cy + r), (cx - r, cy)]
        if filled: d.polygon(pts, fill=color)
        else: d.line(pts + [pts[0]], fill=color, width=width, joint="curve")
    elif kind == "triangle":
        pts = [(cx, cy - r), (cx + r * 0.92, cy + r * 0.72), (cx - r * 0.92, cy + r * 0.72)]
        if filled: d.polygon(pts, fill=color)
        else: d.line(pts + [pts[0]], fill=color, width=width, joint="curve")
    elif kind == "hex":
        pts = [(cx + r * math.cos(math.pi / 3 * k - math.pi / 6),
                cy + r * math.sin(math.pi / 3 * k - math.pi / 6)) for k in range(6)]
        if filled: d.polygon(pts, fill=color)
        else: d.line(pts + [pts[0]], fill=color, width=width, joint="curve")
    elif kind == "asterisk":
        for k in range(3):
            a = math.pi / 3 * k
            d.line([(cx - r * math.cos(a), cy - r * math.sin(a)),
                    (cx + r * math.cos(a), cy + r * math.sin(a))], fill=color, width=width)
    elif kind == "dots":
        for dx, dy in [(-1, 0), (0, 0), (1, 0), (0, -1), (0, 1)]:
            rr = r * 0.20
            d.ellipse([cx + dx * r * 0.5 - rr, cy + dy * r * 0.5 - rr,
                       cx + dx * r * 0.5 + rr, cy + dy * r * 0.5 + rr], fill=color)

KINDS = ["circle", "square", "diamond", "triangle", "hex", "asterisk", "dots"]
seq = random.Random(42)
kinds = [seq.choice(KINDS) for _ in range(cols * rows)]

idx = 0
for r in range(rows):
    for c in range(cols):
        cx = GX0 + (c + 0.5) * cw
        cyc = grid_top + (r + 0.5) * ch
        base_y = cyc + ch * 0.30
        # ledger hairline
        x0l, x1l = cx - cw * 0.33, cx + cw * 0.33
        draw.line([(x0l, base_y), (x1l, base_y)], fill=LINE, width=1)
        draw.line([(x0l, base_y - 5), (x0l, base_y + 5)], fill=LINE, width=1)
        draw.line([(x1l, base_y - 5), (x1l, base_y + 5)], fill=LINE, width=1)
        # specimen pair
        sy = cyc - ch * 0.05
        kind = kinds[idx]
        rr = 16
        dx = 30
        a_color = INK
        b_color = INK
        # sparse pigment: rare green equivalents / amber originals
        if idx % 9 == 4:
            b_color = GREEN
        if idx % 13 == 7:
            a_color = AMBER
        draw_shape(draw, cx - dx, sy, rr, kind, True, a_color, 2)
        draw_shape(draw, cx + dx, sy, rr, kind, False, b_color, 2)
        # connector thread with tiny arrow
        draw.line([(cx - dx + rr + 4, sy), (cx + dx - rr - 6, sy)], fill=MUTE, width=1)
        draw.line([(cx + dx - rr - 6, sy), (cx + dx - rr - 12, sy - 3)], fill=MUTE, width=1)
        draw.line([(cx + dx - rr - 6, sy), (cx + dx - rr - 12, sy + 3)], fill=MUTE, width=1)
        # specimen code
        code = f"EQ·{idx:03d}"
        cw_txt = sum(draw.textlength(cc, font=MONO_S) for cc in code) + 1.5 * (len(code) - 1)
        tracked(cx - cw_txt / 2, base_y + 12, code, MONO_S, MUTE, tr=1.5)
        idx += 1

# ---- legend band ----
leg_y = grid_bottom + 44
draw.line([(GX0, leg_y - 20), (GX1, leg_y - 20)], fill=LINE, width=1)
entries = [
    ("OBSERVATION", "what is installed"),
    ("ASSESSMENT", "verified dimensions"),
    ("EQUIVALENT", "free · open · enough"),
    ("RENEWAL", "switch · keep the data"),
]
ew = (GX1 - GX0) / 4
for i, (word, desc) in enumerate(entries):
    ex = GX0 + i * ew
    gx = ex + 20
    gy = leg_y + 20
    # mini concept diagrams
    if i == 0:  # observation: ring cage + dot
        draw.ellipse([gx - 16, gy - 16, gx + 16, gy + 16], outline=INK, width=1)
        draw.ellipse([gx - 5, gy - 5, gx + 5, gy + 5], fill=INK)
    elif i == 1:  # assessment: circle + tick marks
        draw.ellipse([gx - 13, gy - 13, gx + 13, gy + 13], outline=INK, width=2)
        for ang in (-90, -45, 0, 45, 90, 135, 180, 225):
            a = math.radians(ang)
            draw.line([(gx + 14 * math.cos(a), gy + 14 * math.sin(a)),
                       (gx + 21 * math.cos(a), gy + 21 * math.sin(a))], fill=MUTE, width=1)
    elif i == 2:  # equivalent: filled -> ring
        draw.ellipse([gx - 20, gy - 9, gx - 2, gy + 9], fill=AMBER)
        draw.ellipse([gx + 8, gy - 10, gx + 28, gy + 10], outline=GREEN, width=2)
        draw.line([(gx + 0, gy), (gx + 8, gy)], fill=MUTE, width=1)
    else:  # renewal: hollow -> solid
        draw.ellipse([gx - 20, gy - 10, gx, gy + 10], outline=MUTE, width=1)
        draw.ellipse([gx + 6, gy - 11, gx + 30, gy + 11], fill=GREEN)
        draw.line([(gx + 1, gy), (gx + 6, gy)], fill=MUTE, width=1)
    tracked(gx + 44, gy - 8, word, LEG_T, INK, tr=1.8)
    draw.text((gx + 44, gy + 12), desc, font=LEG_S, fill=MUTE)

# ---- footer ----
fy = H - M - 58
draw.line([(GX0, fy - 26), (GX1, fy - 26)], fill=LINE, width=1)
tracked(GX0, fy, "OBSERVED ON THIS DEVICE  —  NOTHING LEAVES IT", MONO_XS, MUTE, tr=2.4)
foot_r = "FOLIO 01 · 01"
wr = sum(draw.textlength(cc, font=MONO_XS) for cc in foot_r) + 2.4 * (len(foot_r) - 1)
tracked(GX1 - wr, fy, foot_r, MONO_XS, MUTE, tr=2.4)

# ---- CJK margin inscription (subtle thread) ----
try:
    cjk = ImageFont.truetype("/System/Library/Fonts/Supplemental/Songti.ttc", 28, index=0)
    phrase = "看 见 评 估 替 换"
    cxm = M + 38
    n = len(phrase.replace(" ", ""))
    chars = [c for c in phrase if c != " "]
    starty = grid_top + (grid_bottom - grid_top) / 2 - (len(chars) * 42) / 2
    for i, chc in enumerate(chars):
        draw.text((cxm, starty + i * 42), chc, font=cjk, fill=(120, 116, 104), anchor="ma")
except Exception as e:
    print("CJK skipped:", e)

# ---- output ----
png_path = f"{OUT}/cabinet-of-equivalents.png"
pdf_path = f"{OUT}/cabinet-of-equivalents.pdf"
img.save(png_path, "PNG")
img.save(pdf_path, "PDF", resolution=300.0)
print("saved:", png_path, pdf_path, img.size)
