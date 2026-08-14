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
    return authMessages[error.name] ?? error.message;
  }
  return 'Something went wrong. Please try again.';
}
