// Centralized icon sizes (px) for lucide-solid `size` props.
// Names are size labels, not role labels — pick the px that fits the spot.
// Bumping a value here shifts every call site using that token. See
// docs/theming.md §"Icon sizing" for example uses.
export const ICON = {
  xs: 14,
  sm: 15,
  md: 16,
  lg: 18,
  xl: 20,
  xxl: 22,
} as const;
