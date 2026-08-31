type Color = [number, number, number, number]

interface Palette {
  surface: Color
  surfaceAlt: Color
  shadow: Color
  detail: Color
  primary: Color
  cyan: Color
  green: Color
  coral: Color
  white: Color
}

interface CardOptions {
  index: number
  name: string
  x: number
  y: number
  width: number
  height: number
  radius: number
  float: number
  background: Color
  accent: Color
  detail: Color
  shadow: Color
  central?: boolean
}

const color = (hex: string): Color => {
  const value = Number.parseInt(hex.replace('#', ''), 16)
  return [((value >> 16) & 255) / 255, ((value >> 8) & 255) / 255, (value & 255) / 255, 1]
}

const value = (current: unknown) => ({ a: 0, k: current })
const easeIn = { x: [0.42], y: [1] }
const easeOut = { x: [0.58], y: [0] }

const animatedPosition = (x: number, y: number, distance: number) => ({
  a: 1,
  k: [
    { t: 0, s: [x, y, 0], e: [x, y - distance, 0], i: easeIn, o: easeOut },
    { t: 60, s: [x, y - distance, 0], e: [x, y, 0], i: easeIn, o: easeOut },
    { t: 120, s: [x, y, 0] },
  ],
})

const animatedScale = () => ({
  a: 1,
  k: [
    { t: 0, s: [98, 98, 100], e: [103, 103, 100], i: easeIn, o: easeOut },
    { t: 60, s: [103, 103, 100], e: [98, 98, 100], i: easeIn, o: easeOut },
    { t: 120, s: [98, 98, 100] },
  ],
})

const transform = () => ({
  ty: 'tr',
  p: value([0, 0]),
  a: value([0, 0]),
  s: value([100, 100]),
  r: value(0),
  o: value(100),
  sk: value(0),
  sa: value(0),
})

const roundedRect = (
  name: string,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
  fill: Color,
  opacity = 100,
) => ({
  ty: 'gr',
  nm: name,
  it: [
    { ty: 'rc', d: 1, s: value([width, height]), p: value([x, y]), r: value(radius) },
    { ty: 'fl', c: value(fill), o: value(opacity), r: 1 },
    transform(),
  ],
})

const circle = (name: string, x: number, y: number, size: number, fill: Color, opacity = 100) => ({
  ty: 'gr',
  nm: name,
  it: [
    { ty: 'el', d: 1, s: value([size, size]), p: value([x, y]) },
    { ty: 'fl', c: value(fill), o: value(opacity), r: 1 },
    transform(),
  ],
})

const cardLayer = (options: CardOptions) => {
  const { width, height, radius, accent, detail } = options
  const iconX = -width / 2 + 15
  const barX = iconX + 25
  const coreScale = Math.min(width / 68, height / 58)
  const shapes: object[] = [
    roundedRect('投影', 0, 3, width, height, radius, options.shadow, 12),
    roundedRect('卡片', 0, 0, width, height, radius, options.background),
  ]

  if (options.central === true) {
    shapes.push(
      roundedRect('核心模块', 0, 0, width, height, radius, accent),
      roundedRect(
        '模块一',
        -13 * coreScale,
        -11 * coreScale,
        14 * coreScale,
        12 * coreScale,
        4 * coreScale,
        options.detail,
      ),
      roundedRect(
        '模块二',
        8 * coreScale,
        -11 * coreScale,
        18 * coreScale,
        12 * coreScale,
        4 * coreScale,
        options.detail,
        78,
      ),
      roundedRect(
        '模块三',
        -10 * coreScale,
        10 * coreScale,
        20 * coreScale,
        12 * coreScale,
        4 * coreScale,
        options.detail,
        88,
      ),
      roundedRect(
        '模块四',
        14 * coreScale,
        10 * coreScale,
        10 * coreScale,
        12 * coreScale,
        4 * coreScale,
        options.detail,
        66,
      ),
    )
  } else {
    shapes.push(
      roundedRect('图标', iconX, 0, 18, 18, 5, accent),
      circle('图标状态', iconX, 0, 5, options.background),
      roundedRect('信息一', barX, -4, Math.max(16, width - 49), 4, 2, detail, 68),
      roundedRect('信息二', barX - 5, 5, Math.max(12, width - 59), 3, 1.5, detail, 38),
    )
  }

  return {
    ddd: 0,
    ind: options.index,
    ty: 4,
    nm: options.name,
    sr: 1,
    ks: {
      o: value(100),
      r: value(0),
      p: animatedPosition(options.x, options.y, options.float),
      a: value([0, 0, 0]),
      s: options.central === true ? animatedScale() : value([100, 100, 100]),
    },
    ao: 0,
    shapes: shapes.reverse(),
    ip: 0,
    op: 120,
    st: 0,
    bm: 0,
  }
}

const statusLayer = (
  index: number,
  name: string,
  x: number,
  y: number,
  size: number,
  fill: Color,
  square = false,
) => ({
  ddd: 0,
  ind: index,
  ty: 4,
  nm: name,
  sr: 1,
  ks: {
    o: {
      a: 1,
      k: [
        { t: 0, s: [35], e: [100], i: easeIn, o: easeOut },
        { t: 60, s: [100], e: [35], i: easeIn, o: easeOut },
        { t: 120, s: [35] },
      ],
    },
    r: value(0),
    p: animatedPosition(x, y, 3),
    a: value([0, 0, 0]),
    s: value([100, 100, 100]),
  },
  ao: 0,
  shapes: [
    square
      ? roundedRect('状态', 0, 0, size, size, 2, fill)
      : circle('状态', 0, 0, size, fill),
  ],
  ip: 0,
  op: 120,
  st: 0,
  bm: 0,
})

const getPalette = (dark: boolean): Palette => ({
  surface: color(dark ? '#292e3d' : '#ffffff'),
  surfaceAlt: color(dark ? '#33394a' : '#f3f5fa'),
  shadow: color(dark ? '#090b10' : '#59657a'),
  detail: color(dark ? '#cbd5e1' : '#657188'),
  primary: color(dark ? '#8f83f6' : '#7164ed'),
  cyan: color(dark ? '#57b8ca' : '#2d9fb5'),
  green: color(dark ? '#59c396' : '#3aaa7c'),
  coral: color(dark ? '#e28c81' : '#d9796d'),
  white: color('#ffffff'),
})

export const createLoginFlowAnimation = (dark: boolean) => {
  const palette = getPalette(dark)

  return {
    v: '5.12.2',
    fr: 30,
    ip: 0,
    op: 120,
    w: 460,
    h: 112,
    nm: '登录模块协作',
    ddd: 0,
    assets: [],
    layers: [
      statusLayer(8, '左侧状态', 39, 68, 8, palette.coral),
      statusLayer(7, '右侧状态', 426, 47, 9, palette.cyan, true),
      cardLayer({
        index: 6,
        name: '左上模块',
        x: 99,
        y: 30,
        width: 92,
        height: 38,
        radius: 10,
        float: 4,
        background: palette.surface,
        accent: palette.cyan,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 5,
        name: '左下模块',
        x: 130,
        y: 84,
        width: 76,
        height: 32,
        radius: 9,
        float: -3,
        background: palette.surfaceAlt,
        accent: palette.green,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 4,
        name: '右上模块',
        x: 342,
        y: 30,
        width: 78,
        height: 36,
        radius: 10,
        float: -4,
        background: palette.surfaceAlt,
        accent: palette.coral,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 3,
        name: '右下模块',
        x: 362,
        y: 84,
        width: 96,
        height: 34,
        radius: 9,
        float: 3,
        background: palette.surface,
        accent: palette.cyan,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 2,
        name: '平台核心',
        x: 230,
        y: 57,
        width: 68,
        height: 58,
        radius: 14,
        float: 3,
        background: palette.primary,
        accent: palette.primary,
        detail: palette.white,
        shadow: palette.shadow,
        central: true,
      }),
      statusLayer(1, '核心状态', 276, 26, 8, palette.green),
    ],
  }
}

export const createLoginPlatformAnimation = (dark: boolean) => {
  const palette = getPalette(dark)

  return {
    v: '5.12.2',
    fr: 30,
    ip: 0,
    op: 120,
    w: 760,
    h: 520,
    nm: '平台模块协作',
    ddd: 0,
    assets: [],
    layers: [
      statusLayer(12, '左侧状态', 63, 244, 13, palette.coral),
      statusLayer(11, '右侧状态', 707, 260, 14, palette.cyan, true),
      statusLayer(10, '顶部状态', 430, 68, 12, palette.green),
      statusLayer(9, '底部状态', 326, 460, 10, palette.primary, true),
      cardLayer({
        index: 8,
        name: '身份模块',
        x: 172,
        y: 137,
        width: 164,
        height: 66,
        radius: 16,
        float: 9,
        background: palette.surface,
        accent: palette.cyan,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 7,
        name: '组织模块',
        x: 134,
        y: 304,
        width: 138,
        height: 58,
        radius: 14,
        float: -7,
        background: palette.surfaceAlt,
        accent: palette.green,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 6,
        name: '数据模块',
        x: 232,
        y: 423,
        width: 172,
        height: 64,
        radius: 15,
        float: 8,
        background: palette.surface,
        accent: palette.coral,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 5,
        name: '配置模块',
        x: 583,
        y: 133,
        width: 150,
        height: 62,
        radius: 15,
        float: -8,
        background: palette.surfaceAlt,
        accent: palette.coral,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 4,
        name: '集成模块',
        x: 630,
        y: 298,
        width: 164,
        height: 66,
        radius: 16,
        float: 9,
        background: palette.surface,
        accent: palette.cyan,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 3,
        name: '审计模块',
        x: 542,
        y: 424,
        width: 146,
        height: 60,
        radius: 14,
        float: -7,
        background: palette.surfaceAlt,
        accent: palette.green,
        detail: palette.detail,
        shadow: palette.shadow,
      }),
      cardLayer({
        index: 2,
        name: '平台核心',
        x: 380,
        y: 260,
        width: 126,
        height: 108,
        radius: 25,
        float: 7,
        background: palette.primary,
        accent: palette.primary,
        detail: palette.white,
        shadow: palette.shadow,
        central: true,
      }),
      statusLayer(1, '核心状态', 466, 215, 13, palette.green),
    ],
  }
}
