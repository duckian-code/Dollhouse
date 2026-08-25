import { StyleSheet, Text, View } from 'react-native';

import { tokens } from '@/theme/tokens';

type SectionHeaderProps = {
  title: string;
  subtitle?: string;
};

export function SectionHeader({ title, subtitle }: SectionHeaderProps) {
  return (
    <View style={styles.container}>
      <Text accessibilityRole="header" style={styles.title}>
        {title}
      </Text>
      {subtitle ? <Text style={styles.subtitle}>{subtitle}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { marginBottom: tokens.spacing.lg },
  title: {
    color: tokens.color.text,
    fontSize: tokens.typography.title,
    fontWeight: '700',
    letterSpacing: -0.6,
  },
  subtitle: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontSize: tokens.typography.body,
    lineHeight: 23,
  },
});
