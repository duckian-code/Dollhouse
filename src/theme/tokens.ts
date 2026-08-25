export const tokens = {
  color: {
    background: '#F8F1E8',
    surface: '#FFFDFC',
    surfaceMuted: '#EFE2D3',
    text: '#2F2520',
    textMuted: '#715849',
    accent: '#7A4E38',
    accentSoft: '#E8D1B8',
    border: '#E7D9CA',
    white: '#FFFFFF',
  },
  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
  },
  radius: {
    sm: 10,
    md: 16,
    lg: 24,
    round: 999,
  },
  typography: {
    title: 32,
    heading: 20,
    body: 16,
    caption: 13,
  },
  shadow: {
    card: {
      shadowColor: '#2F2520',
      shadowOffset: { width: 0, height: 4 },
      shadowOpacity: 0.08,
      shadowRadius: 12,
      elevation: 2,
    },
  },
} as const;
