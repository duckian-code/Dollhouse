import { StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { SectionHeader } from '@/components/SectionHeader';
import { tokens } from '@/theme/tokens';

const placeholders = [
  ['Today', 'No entries yet'],
  ['This week', 'Your mood chart will appear here'],
  ['Patterns', 'Insights are coming in a later update'],
] as const;

export default function MoodsScreen() {
  return (
    <AppScreen>
      <SectionHeader
        title="Moods"
        subtitle="Your check-in history and patterns."
      />
      <View style={styles.list}>
        {placeholders.map(([title, detail]) => (
          <AppCard key={title}>
            <Text style={styles.title}>{title}</Text>
            <Text style={styles.detail}>{detail}</Text>
          </AppCard>
        ))}
      </View>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  list: { gap: tokens.spacing.md },
  title: { color: tokens.color.text, fontSize: 17, fontWeight: '700' },
  detail: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontSize: 14,
  },
});
