import { Tabs } from 'expo-router';
import { type ColorValue, StyleSheet, Text } from 'react-native';

import { tokens } from '@/theme/tokens';

const icons: Record<string, string> = {
  index: '⌂',
  moods: '◉',
  friends: '♧',
  profile: '○',
};

function TabIcon({ color, name }: { color: ColorValue; name: string }) {
  return <Text style={[styles.icon, { color }]}>{icons[name]}</Text>;
}

export default function TabLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: tokens.color.accent,
        tabBarInactiveTintColor: tokens.color.textMuted,
        tabBarLabelStyle: styles.label,
        tabBarStyle: styles.tabBar,
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Home',
          tabBarIcon: ({ color }) => <TabIcon color={color} name="index" />,
        }}
      />
      <Tabs.Screen
        name="moods"
        options={{
          title: 'Moods',
          tabBarIcon: ({ color }) => <TabIcon color={color} name="moods" />,
        }}
      />
      <Tabs.Screen
        name="friends"
        options={{
          title: 'Friends',
          tabBarIcon: ({ color }) => <TabIcon color={color} name="friends" />,
        }}
      />
      <Tabs.Screen
        name="profile"
        options={{
          title: 'Profile',
          tabBarIcon: ({ color }) => <TabIcon color={color} name="profile" />,
        }}
      />
    </Tabs>
  );
}

const styles = StyleSheet.create({
  tabBar: {
    minHeight: 68,
    paddingTop: tokens.spacing.sm,
    paddingBottom: tokens.spacing.sm,
    borderTopColor: tokens.color.border,
    backgroundColor: tokens.color.surface,
  },
  label: { fontFamily: tokens.typography.headingSemibold, fontSize: 12 },
  icon: {
    fontFamily: tokens.typography.headingRegular,
    fontSize: 24,
    lineHeight: 26,
  },
});
