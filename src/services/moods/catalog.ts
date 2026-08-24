export interface MoodOption {
  status: string;
  label: string;
  symbol: string;
}

// The backend accepts an open status string rather than a mood ID. Keeping the
// catalog here gives the UI a consistent set of choices without inventing an
// API route that does not exist in the backend contract.
export const moodCatalog: readonly MoodOption[] = [
  { status: 'Happy', label: 'Happy', symbol: '☀' },
  { status: 'Calm', label: 'Calm', symbol: '◡' },
  { status: 'Content', label: 'Content', symbol: '◇' },
  { status: 'Sad', label: 'Sad', symbol: '☂' },
  { status: 'Anxious', label: 'Anxious', symbol: '≈' },
  { status: 'Angry', label: 'Angry', symbol: '⚡' },
  { status: 'Overwhelmed', label: 'Overwhelmed', symbol: '✦' },
  { status: 'Tired', label: 'Tired', symbol: '☾' },
];

export function isCatalogMood(status: string | null): status is string {
  return moodCatalog.some((mood) => mood.status === status);
}
