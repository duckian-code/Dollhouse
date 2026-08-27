const authMessages: Record<string, string> = {
  CodeMismatchException: 'That confirmation code is not correct.',
  InvalidPasswordException:
    'Use at least 12 characters with uppercase, lowercase, a number, and a symbol.',
  LimitExceededException: 'Too many attempts. Please wait and try again.',
  NotAuthorizedException: 'The email or password is incorrect.',
  UsernameExistsException: 'An account already exists for this email.',
  UserNotConfirmedException: 'Confirm your email before signing in.',
  UserNotFoundException: 'The email or password is incorrect.',
};

export function getAuthErrorMessage(error: unknown) {
  if (error instanceof Error) {
    if (authMessages[error.name]) return authMessages[error.name];
    if (error.name === 'NetworkError' || error instanceof TypeError) {
      return 'You appear to be offline. Check your connection and try again.';
    }
    if (error.message.includes('not configured')) {
      return 'Sign-in is temporarily unavailable because authentication has not been configured.';
    }
    return 'We could not complete that request. Please try again.';
  }
  return 'Something went wrong. Please try again.';
}
