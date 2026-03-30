#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = [
#     "pygerber",
#     "trimesh",
#     "numpy",
#     "pillow",
#     "scipy",
#     "mapbox-earcut",
#     "manifold3d",
#     "shapely",
# ]
# ///
"""Generate a GLB 3D model from PCB Gerber files.

Converts a set of Gerber manufacturing files into a textured 3D GLB model with:
- Raster-composited PBR textures (base color + metallic/roughness)
- Board cutouts from Edge_Cuts and NPTH drill files
- Optional component import from an existing GLB

Usage:
    uv run generate.py <gerber_dir> [options]

Examples:
    uv run generate.py ./gerber_files -o board.glb
    uv run generate.py ./gerber_files --dpmm 40 --components design.glb
    uv run generate.py ./gerber_files --thickness 0.8
"""

import argparse
import io
import os
import re
import sys

import numpy as np
from PIL import Image
from pygerber.gerberx3.api.v2 import GerberFile, ColorScheme
from pygerber.gerberx3.api._v2 import PixelFormatEnum, ImageFormatEnum
from pygerber.common.rgba import RGBA
import trimesh
import trimesh.visual
import trimesh.visual.material


# --- Color schemes ---

SOLDER_MASK_COLORS = {
    "black": ColorScheme(
        background_color=RGBA(r=6, g=6, b=6, a=250),
        clear_color=RGBA(r=0, g=0, b=0, a=0),
        solid_color=RGBA(r=0, g=0, b=0, a=0),
        clear_region_color=RGBA(r=0, g=0, b=0, a=0),
        solid_region_color=RGBA(r=0, g=0, b=0, a=0),
    ),
    "green": ColorScheme(
        background_color=RGBA(r=0, g=60, b=20, a=240),
        clear_color=RGBA(r=0, g=0, b=0, a=0),
        solid_color=RGBA(r=0, g=0, b=0, a=0),
        clear_region_color=RGBA(r=0, g=0, b=0, a=0),
        solid_region_color=RGBA(r=0, g=0, b=0, a=0),
    ),
    "blue": ColorScheme(
        background_color=RGBA(r=0, g=20, b=80, a=240),
        clear_color=RGBA(r=0, g=0, b=0, a=0),
        solid_color=RGBA(r=0, g=0, b=0, a=0),
        clear_region_color=RGBA(r=0, g=0, b=0, a=0),
        solid_region_color=RGBA(r=0, g=0, b=0, a=0),
    ),
    "red": ColorScheme(
        background_color=RGBA(r=120, g=0, b=0, a=240),
        clear_color=RGBA(r=0, g=0, b=0, a=0),
        solid_color=RGBA(r=0, g=0, b=0, a=0),
        clear_region_color=RGBA(r=0, g=0, b=0, a=0),
        solid_region_color=RGBA(r=0, g=0, b=0, a=0),
    ),
    "white": ColorScheme(
        background_color=RGBA(r=230, g=230, b=230, a=240),
        clear_color=RGBA(r=0, g=0, b=0, a=0),
        solid_color=RGBA(r=0, g=0, b=0, a=0),
        clear_region_color=RGBA(r=0, g=0, b=0, a=0),
        solid_region_color=RGBA(r=0, g=0, b=0, a=0),
    ),
}

COPPER_COLORS = ColorScheme(
    background_color=RGBA(r=0, g=0, b=0, a=0),
    clear_color=RGBA(r=0, g=0, b=0, a=0),
    solid_color=RGBA(r=220, g=185, b=50, a=255),
    clear_region_color=RGBA(r=0, g=0, b=0, a=0),
    solid_region_color=RGBA(r=220, g=185, b=50, a=255),
)

SILK_COLORS = ColorScheme(
    background_color=RGBA(r=0, g=0, b=0, a=0),
    clear_color=RGBA(r=0, g=0, b=0, a=0),
    solid_color=RGBA(r=245, g=240, b=225, a=255),
    clear_region_color=RGBA(r=0, g=0, b=0, a=0),
    solid_region_color=RGBA(r=245, g=240, b=225, a=255),
)

WHITE_MASK = ColorScheme(
    background_color=RGBA(r=0, g=0, b=0, a=0),
    clear_color=RGBA(r=0, g=0, b=0, a=0),
    solid_color=RGBA(r=255, g=255, b=255, a=255),
    clear_region_color=RGBA(r=0, g=0, b=0, a=0),
    solid_region_color=RGBA(r=255, g=255, b=255, a=255),
)

MAT_COPPER_EXPOSED = [0.9, 0.25]
MAT_SOLDER_MASK = [0.0, 0.55]
MAT_SILKSCREEN = [0.0, 0.90]
MAT_FR4 = [0.0, 0.70]


# --- Gerber file auto-detection ---

GERBER_PATTERNS = {
    "f_cu":   [r"-F_Cu\.gtl$", r"-F\.Cu\.gtl$", r"\.GTL$"],
    "b_cu":   [r"-B_Cu\.gbl$", r"-B\.Cu\.gbl$", r"\.GBL$"],
    "f_mask": [r"-F_Mask\.gts$", r"-F\.Mask\.gts$", r"\.GTS$"],
    "b_mask": [r"-B_Mask\.gbs$", r"-B\.Mask\.gbs$", r"\.GBS$"],
    "f_silk": [r"-F_Silkscreen\.gto$", r"-F_SilkS\.gto$", r"\.GTO$"],
    "b_silk": [r"-B_Silkscreen\.gbo$", r"-B_SilkS\.gbo$", r"\.GBO$"],
    "edge":   [r"-Edge_Cuts\.gm1$", r"-Edge\.Cuts\.gm1$", r"\.GM1$", r"\.GKO$"],
    "npth":   [r"-NPTH\.drl$", r"\.NPTH\.drl$"],
}


def find_gerber_files(gerber_dir):
    """Auto-detect gerber files in a directory by filename patterns."""
    files = os.listdir(gerber_dir)
    found = {}
    for key, patterns in GERBER_PATTERNS.items():
        for f in files:
            for pat in patterns:
                if re.search(pat, f, re.IGNORECASE):
                    found[key] = os.path.join(gerber_dir, f)
                    break
            if key in found:
                break
    return found


# --- Rendering ---

def render_gerber(filepath, color_scheme, dpmm):
    buf = io.BytesIO()
    gf = GerberFile.from_file(filepath)
    parsed = gf.parse()
    info = parsed.get_info()
    parsed.render_raster(
        buf, color_scheme=color_scheme, dpmm=dpmm,
        image_format=ImageFormatEnum.PNG, pixel_format=PixelFormatEnum.RGBA,
    )
    buf.seek(0)
    return Image.open(buf).copy(), info


def crop_to_board(img, info, board_bounds, dpmm):
    min_x = float(info.min_x_mm)
    max_y = float(info.max_y_mm)
    bmin_x, bmin_y, bmax_x, bmax_y = board_bounds
    left = max(0, int((bmin_x - min_x) * dpmm))
    right = min(img.width, int((bmax_x - min_x) * dpmm))
    top = max(0, int((max_y - bmax_y) * dpmm))
    bottom = min(img.height, int((max_y - bmin_y) * dpmm))
    return img.crop((left, top, right, bottom))


def composite_face(layers):
    w, h = layers[0].size
    result = Image.new("RGBA", (w, h), (25, 25, 20, 255))
    for layer in layers:
        lyr = layer.resize((w, h), Image.Resampling.LANCZOS) if layer.size != (w, h) else layer
        result = Image.alpha_composite(result, lyr)
    return result.convert("RGB")


def build_metallic_roughness(copper_mask, solder_mask, silk_mask, size):
    w, h = size
    def to_array(img):
        return np.array(img.resize((w, h), Image.Resampling.LANCZOS).convert("L"), dtype=np.float32) / 255.0
    copper_a = to_array(copper_mask)
    mask_a = to_array(solder_mask)
    silk_a = to_array(silk_mask)

    metallic = np.full((h, w), MAT_FR4[0], dtype=np.float32)
    roughness = np.full((h, w), MAT_FR4[1], dtype=np.float32)

    for mat_vals, blend in [(MAT_SOLDER_MASK, mask_a), (MAT_SILKSCREEN, silk_a), (MAT_COPPER_EXPOSED, copper_a)]:
        metallic = metallic * (1 - blend) + mat_vals[0] * blend
        roughness = roughness * (1 - blend) + mat_vals[1] * blend

    mr_img = np.zeros((h, w, 3), dtype=np.uint8)
    mr_img[:, :, 0] = 255
    mr_img[:, :, 1] = np.clip(roughness * 255, 0, 255).astype(np.uint8)
    mr_img[:, :, 2] = np.clip(metallic * 255, 0, 255).astype(np.uint8)
    return Image.fromarray(mr_img, "RGB")


# --- Board geometry ---

def parse_edge_cuts(filepath):
    paths = []
    current_path = []
    current_x, current_y = 0.0, 0.0
    scale = 1e-6

    with open(filepath) as f:
        for line in f:
            line = line.strip().rstrip("*")
            if not line or line.startswith("G04") or line.startswith("%"):
                continue
            if line.startswith("D") and line[1:].isdigit():
                if current_path and len(current_path) >= 2:
                    paths.append(current_path)
                current_path = []
                continue
            m = re.match(r'X(-?\d+)Y(-?\d+)D(\d+)', line)
            if m:
                x = float(m.group(1)) * scale
                y = float(m.group(2)) * scale
                d = int(m.group(3))
                if d == 2:
                    if current_path:
                        last = current_path[-1]
                        if abs(last[0] - x) < 0.01 and abs(last[1] - y) < 0.01:
                            continue
                    if current_path and len(current_path) >= 2:
                        paths.append(current_path)
                    current_path = [(x, y)]
                elif d == 1:
                    if not current_path:
                        current_path = [(current_x, current_y)]
                    current_path.append((x, y))
                current_x, current_y = x, y
    if current_path and len(current_path) >= 2:
        paths.append(current_path)

    from shapely.geometry import Polygon
    polygons = []
    for path in paths:
        if len(path) < 3:
            continue
        dx = abs(path[0][0] - path[-1][0])
        dy = abs(path[0][1] - path[-1][1])
        if dx < 0.01 and dy < 0.01:
            poly = Polygon(path)
            if poly.is_valid and poly.area > 0.1:
                polygons.append(poly)
    if not polygons:
        return None, []
    polygons.sort(key=lambda p: p.area, reverse=True)
    return polygons[0], polygons[1:]


def parse_npth_holes(filepath):
    holes = []
    current_diameter = 0
    is_inch = False
    tools = {}
    with open(filepath) as f:
        for line in f:
            line = line.strip()
            if line == "INCH":
                is_inch = True
            elif line == "METRIC":
                is_inch = False
            elif line.startswith("T") and "C" in line and not line.startswith("T0"):
                parts = line.split("C")
                diameter = float(parts[1])
                if is_inch:
                    diameter *= 25.4
                tools[parts[0]] = diameter
            elif line.startswith("T") and line[1:].isdigit():
                current_diameter = tools.get(line, 0)
            elif line.startswith("X") and "Y" in line and current_diameter > 0:
                x_str = line.split("Y")[0][1:]
                y_str = line.split("Y")[1]
                x, y = float(x_str), float(y_str)
                if is_inch:
                    x *= 25.4
                    y *= 25.4
                holes.append((x, y, current_diameter))
    return holes


def get_board_bounds(edge_file):
    """Get board bounds from Edge_Cuts file. Returns (min_x, min_y, max_x, max_y)."""
    outline, _ = parse_edge_cuts(edge_file)
    if outline:
        b = outline.bounds
        return b[0], b[1], b[2], b[3]
    return None


def build_board_polygon(gerber_files, board_bounds):
    from shapely.geometry import Point, box as shapely_box
    from shapely.affinity import translate
    from shapely.ops import unary_union

    bmin_x, bmin_y, bmax_x, bmax_y = board_bounds
    board_cx = (bmin_x + bmax_x) / 2
    board_cy = (bmin_y + bmax_y) / 2

    def to_model(polygon):
        return translate(polygon, -board_cx, -board_cy)

    if "edge" in gerber_files:
        outline, poly_cutouts = parse_edge_cuts(gerber_files["edge"])
        if outline:
            board = to_model(outline)
            print(f"  Board outline: {board.area:.0f} mm²")
        else:
            hw = (bmax_x - bmin_x) / 2
            hh = (bmax_y - bmin_y) / 2
            board = shapely_box(-hw, -hh, hw, hh)
            print("  Board outline: fallback rectangle")
            poly_cutouts = []
    else:
        hw = (bmax_x - bmin_x) / 2
        hh = (bmax_y - bmin_y) / 2
        board = shapely_box(-hw, -hh, hw, hh)
        poly_cutouts = []

    cutout_list = []
    for c in poly_cutouts:
        mc = to_model(c)
        cutout_list.append(mc)
        b = mc.bounds
        print(f"  Polygon cutout: {b[2]-b[0]:.1f}x{b[3]-b[1]:.1f}mm")

    if "npth" in gerber_files:
        holes = parse_npth_holes(gerber_files["npth"])
        for x, y, d in holes:
            circle = to_model(Point(x, y).buffer(d / 2, resolution=32))
            cutout_list.append(circle)
        print(f"  Drill holes: {len(holes)}")

    if cutout_list:
        board = board.difference(unary_union(cutout_list))

    print(f"  Final board: {board.area:.0f} mm² with {len(cutout_list)} cutouts")
    return board


def create_board(board_poly, board_bounds, thickness, top_tex, bot_tex, top_mr, bot_mr):
    bmin_x, bmin_y, bmax_x, bmax_y = board_bounds
    hw = (bmax_x - bmin_x) / 2
    hh = (bmax_y - bmin_y) / 2
    t = thickness
    scene = trimesh.Scene()

    board_mesh = trimesh.creation.extrude_polygon(board_poly, t)

    # Strip top/bottom cap faces — we add our own textured caps below.
    # Keep only side faces (normals that aren't purely +Z or -Z).
    normals = board_mesh.face_normals
    side_mask = np.abs(normals[:, 2]) < 0.99
    board_mesh.update_faces(side_mask)
    board_mesh.remove_unreferenced_vertices()

    edge_color = [50, 55, 35, 255]
    board_mesh.visual = trimesh.visual.ColorVisuals(
        mesh=board_mesh, face_colors=np.tile(edge_color, (len(board_mesh.faces), 1)),
    )
    scene.add_geometry(board_mesh, node_name="board_body")

    # Offset textured caps slightly so they sit above/below any coplanar
    # component geometry (imported parts have faces at exactly Z=0 and Z≈t).
    # 0.09mm is invisible but survives even aggressive Draco quantization.
    cap_eps = 0.09

    top_verts_2d, top_faces_2d = trimesh.creation.triangulate_polygon(board_poly)
    top_verts_3d = np.column_stack([top_verts_2d, np.full(len(top_verts_2d), t + cap_eps)])
    top_mesh = trimesh.Trimesh(vertices=top_verts_3d, faces=top_faces_2d, process=False)
    top_uv = np.column_stack([
        (top_verts_2d[:, 0] + hw) / (2 * hw),
        (top_verts_2d[:, 1] + hh) / (2 * hh),
    ])
    top_mesh.visual = trimesh.visual.TextureVisuals(
        uv=top_uv,
        material=trimesh.visual.material.PBRMaterial(
            baseColorTexture=top_tex, metallicRoughnessTexture=top_mr,
            metallicFactor=1.0, roughnessFactor=1.0,
        ),
    )
    scene.add_geometry(top_mesh, node_name="top_face")

    bot_verts_3d = np.column_stack([top_verts_2d, np.full(len(top_verts_2d), -cap_eps)])
    bot_mesh = trimesh.Trimesh(vertices=bot_verts_3d, faces=top_faces_2d[:, ::-1], process=False)
    bot_uv = np.column_stack([
        1.0 - (top_verts_2d[:, 0] + hw) / (2 * hw),
        (top_verts_2d[:, 1] + hh) / (2 * hh),
    ])
    bot_mesh.visual = trimesh.visual.TextureVisuals(
        uv=bot_uv,
        material=trimesh.visual.material.PBRMaterial(
            baseColorTexture=bot_tex, metallicRoughnessTexture=bot_mr,
            metallicFactor=1.0, roughnessFactor=1.0,
        ),
    )
    scene.add_geometry(bot_mesh, node_name="bottom_face")

    return scene


def import_components(scene, glb_path, board_bounds):
    """Import component meshes from an existing GLB, transforming to board-centered coords."""
    if not os.path.exists(glb_path):
        print(f"  Component GLB not found: {glb_path}")
        return

    print(f"Importing components from {glb_path}...")
    src = trimesh.load(glb_path)
    assert isinstance(src, trimesh.Scene)

    bmin_x, bmin_y, bmax_x, bmax_y = board_bounds
    board_cx = (bmin_x + bmax_x) / 2
    board_cy = (bmin_y + bmax_y) / 2

    # Detect coordinate system from the GLB bounds
    src_bounds = src.bounds
    src_size = src_bounds[1] - src_bounds[0]

    # If bounds are < 1 in all dims, assume meters; otherwise mm
    if max(src_size) < 1.0:
        scale = 1000.0  # meters to mm
    else:
        scale = 1.0

    # Detect axis mapping: check which axis pair matches board width/height
    bw = bmax_x - bmin_x
    bh = bmax_y - bmin_y
    sx, sy, sz = src_size * scale

    # Try X-Z mapping (common in STEP/GLB exports)
    if abs(sx - bw) < 5 and abs(sz - bh) < 5:
        transform = np.array([
            [scale, 0, 0, -board_cx],
            [0, 0, -scale, -board_cy],
            [0, scale, 0, 0],
            [0, 0, 0, 1],
        ], dtype=float)
    # Try X-Y mapping
    elif abs(sx - bw) < 5 and abs(sy - bh) < 5:
        transform = np.array([
            [scale, 0, 0, -board_cx],
            [0, scale, 0, -board_cy],
            [0, 0, scale, 0],
            [0, 0, 0, 1],
        ], dtype=float)
    else:
        print(f"  Warning: could not auto-detect axis mapping (src size: {sx:.1f}x{sy:.1f}x{sz:.1f}mm, board: {bw:.1f}x{bh:.1f}mm)")
        transform = np.array([
            [scale, 0, 0, -board_cx],
            [0, 0, -scale, -board_cy],
            [0, scale, 0, 0],
            [0, 0, 0, 1],
        ], dtype=float)

    comp_count = 0
    comp_faces = 0
    for node in src.graph.nodes:
        try:
            world_transform, geom_name = src.graph.get(node)
        except (ValueError, KeyError):
            continue
        if geom_name is None or not isinstance(geom_name, str) or geom_name not in src.geometry:
            continue
        if geom_name.startswith("design_soldermask"):
            continue
        if geom_name.startswith("design_PCB"):
            g = src.geometry[geom_name]
            b = g.bounds
            size = b[1] - b[0]
            if max(size[0], size[2]) > 0.1:
                continue
        geom = src.geometry[geom_name]
        if len(geom.faces) == 0:
            continue
        combined = transform @ world_transform
        mesh = geom.copy()
        mesh.apply_transform(combined)
        scene.add_geometry(mesh, node_name=f"comp_{node}")
        comp_count += 1
        comp_faces += len(mesh.faces)

    print(f"  Added {comp_count} component meshes ({comp_faces} faces)")


# --- Main ---

def main():
    parser = argparse.ArgumentParser(description="Generate GLB 3D model from PCB Gerber files")
    parser.add_argument("gerber_dir", help="Directory containing Gerber files")
    parser.add_argument("-o", "--output", default="board.glb", help="Output GLB path (default: board.glb)")
    parser.add_argument("--dpmm", type=int, default=20, help="Texture resolution in dots per mm (default: 20)")
    parser.add_argument("--thickness", type=float, default=1.6, help="Board thickness in mm (default: 1.6)")
    parser.add_argument("--mask-color", choices=list(SOLDER_MASK_COLORS.keys()), default="black",
                        help="Solder mask color (default: black)")
    parser.add_argument("--components", help="Path to GLB file with component models to import")
    parser.add_argument("--save-textures", action="store_true", help="Save intermediate texture PNGs")
    args = parser.parse_args()

    gerber_dir = args.gerber_dir
    if not os.path.isdir(gerber_dir):
        print(f"Error: {gerber_dir} is not a directory", file=sys.stderr)
        sys.exit(1)

    # Auto-detect gerber files
    gerber_files = find_gerber_files(gerber_dir)
    print(f"Found gerber files in {gerber_dir}:")
    for key, path in sorted(gerber_files.items()):
        print(f"  {key}: {os.path.basename(path)}")

    required = ["f_cu", "f_mask", "edge"]
    missing = [k for k in required if k not in gerber_files]
    if missing:
        print(f"Error: missing required gerber files: {missing}", file=sys.stderr)
        sys.exit(1)

    # Get board bounds from edge cuts
    board_bounds = get_board_bounds(gerber_files["edge"])
    if not board_bounds:
        print("Error: could not parse board outline from Edge_Cuts", file=sys.stderr)
        sys.exit(1)

    bw = board_bounds[2] - board_bounds[0]
    bh = board_bounds[3] - board_bounds[1]
    print(f"Board: {bw:.1f} x {bh:.1f} mm, {args.thickness}mm thick")

    mask_scheme = SOLDER_MASK_COLORS[args.mask_color]
    dpmm = args.dpmm
    out_dir = os.path.dirname(os.path.abspath(args.output)) or "."

    def render_and_crop(path, colors, label):
        print(f"  {label}...")
        img, info = render_gerber(path, colors, dpmm)
        return crop_to_board(img, info, board_bounds, dpmm)

    # Render layers
    print("Rendering Gerber layers...")
    f_cu = render_and_crop(gerber_files["f_cu"], COPPER_COLORS, "Front copper")
    f_mask = render_and_crop(gerber_files["f_mask"], mask_scheme, "Front mask")
    f_silk = render_and_crop(gerber_files["f_silk"], SILK_COLORS, "Front silk") if "f_silk" in gerber_files else None
    f_cu_white = render_and_crop(gerber_files["f_cu"], WHITE_MASK, "Front copper mask")
    f_mask_openings = render_and_crop(gerber_files["f_mask"], WHITE_MASK, "Front mask openings")
    f_silk_white = render_and_crop(gerber_files["f_silk"], WHITE_MASK, "Front silk mask") if "f_silk" in gerber_files else None

    front_layers = [f_cu, f_mask]
    if f_silk:
        front_layers.append(f_silk)

    b_cu = render_and_crop(gerber_files["b_cu"], COPPER_COLORS, "Back copper") if "b_cu" in gerber_files else None
    b_mask = render_and_crop(gerber_files["b_mask"], mask_scheme, "Back mask") if "b_mask" in gerber_files else None
    b_silk = render_and_crop(gerber_files["b_silk"], SILK_COLORS, "Back silk") if "b_silk" in gerber_files else None
    b_cu_white = render_and_crop(gerber_files["b_cu"], WHITE_MASK, "Back copper mask") if "b_cu" in gerber_files else None
    b_mask_openings = render_and_crop(gerber_files["b_mask"], WHITE_MASK, "Back mask openings") if "b_mask" in gerber_files else None
    b_silk_white = render_and_crop(gerber_files["b_silk"], WHITE_MASK, "Back silk mask") if "b_silk" in gerber_files else None

    # Composite
    print("Compositing textures...")
    top_texture = composite_face(front_layers)

    back_layers = [layer for layer in [b_cu, b_mask, b_silk] if layer is not None]
    bottom_texture = composite_face(back_layers) if back_layers else top_texture

    if args.save_textures:
        top_texture.save(os.path.join(out_dir, "top_texture.png"))
        bottom_texture.save(os.path.join(out_dir, "bottom_texture.png"))

    # Material maps
    print("Building material maps...")
    size = top_texture.size

    def make_exposed(cu_w, mask_w, sz):
        cu_arr = np.array(cu_w.resize(sz, Image.Resampling.LANCZOS).convert("L"), dtype=np.float32) / 255.0
        mk_arr = np.array(mask_w.resize(sz, Image.Resampling.LANCZOS).convert("L"), dtype=np.float32) / 255.0
        return Image.fromarray((np.minimum(cu_arr, mk_arr) * 255).astype(np.uint8), "L")

    f_exposed = make_exposed(f_cu_white, f_mask_openings, size)
    f_mask_cov = Image.fromarray(255 - np.array(f_mask_openings.resize(size, Image.Resampling.LANCZOS).convert("L")), "L")
    f_silk_m = f_silk_white if f_silk_white else Image.new("L", size, 0)
    top_mr = build_metallic_roughness(f_exposed, f_mask_cov, f_silk_m, size)

    if b_cu_white and b_mask_openings:
        bsize = bottom_texture.size
        b_exposed = make_exposed(b_cu_white, b_mask_openings, bsize)
        b_mask_cov = Image.fromarray(255 - np.array(b_mask_openings.resize(bsize, Image.Resampling.LANCZOS).convert("L")), "L")
        b_silk_m = b_silk_white if b_silk_white else Image.new("L", bsize, 0)
        bot_mr = build_metallic_roughness(b_exposed, b_mask_cov, b_silk_m, bsize)
    else:
        bot_mr = top_mr

    if args.save_textures:
        top_mr.save(os.path.join(out_dir, "top_metallic_roughness.png"))
        bot_mr.save(os.path.join(out_dir, "bottom_metallic_roughness.png"))

    # Board geometry
    print("Building board geometry...")
    board_poly = build_board_polygon(gerber_files, board_bounds)

    print("Building 3D model...")
    scene = create_board(board_poly, board_bounds, args.thickness,
                         top_texture, bottom_texture, top_mr, bot_mr)

    # Components
    if args.components:
        import_components(scene, args.components, board_bounds)

    # Export
    print(f"Exporting to {args.output}...")
    scene.export(args.output, file_type="glb")

    total_faces = sum(g.faces.shape[0] for g in scene.geometry.values())
    size_mb = os.path.getsize(args.output) / 1024 / 1024
    print(f"Total faces: {total_faces}, File size: {size_mb:.1f} MB")
    print("Done!")


if __name__ == "__main__":
    main()
