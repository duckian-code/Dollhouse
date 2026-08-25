import { StyleSheet, Text, View } from 'react-native';

import { AppScreen } from '@/components/AppScreen';

export default function HomeScreen() {
  return (
    <AppScreen>
      <View
        style={styles.house}
        accessible
        accessibilityLabel="Dollhouse app icon"
      >
        <Text style={styles.roof}>⌂</Text>
      </View>
      <Text accessibilityRole="header" style={styles.title}>
        Dollhouse
      </Text>
      <Text style={styles.subtitle}>Application setup complete</Text>
      <Text style={styles.description}>
        Your shared space for checking in, tracking moods, and staying
        connected.
      </Text>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  house: {
    width: 96,
    height: 96,
    borderRadius: 28,
    backgroundColor: '#E8D1B8',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 28,
  },
  roof: {
    color: '#563D2D',
    fontSize: 58,
    lineHeight: 70,
  },
  title: {
    color: '#2F2520',
    fontSize: 36,
    fontWeight: '700',
    letterSpacing: -0.8,
    textAlign: 'center',
  },
  subtitle: {
    color: '#715849',
    fontSize: 17,
    fontWeight: '600',
    marginTop: 8,
    textAlign: 'center',
  },
  description: {
    color: '#715849',
    fontSize: 15,
    lineHeight: 23,
    marginTop: 18,
    maxWidth: 320,
    textAlign: 'center',
  },
});
