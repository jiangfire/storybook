import type { SVGProps } from 'react';
import { cn } from '../../utils/cn';

type IconProps = SVGProps<SVGSVGElement> & {
  size?: number;
};

function BaseIcon({ className, size = 20, children, ...props }: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      width={size}
      height={size}
      className={cn('shrink-0', className)}
      aria-hidden="true"
      {...props}
    >
      {children}
    </svg>
  );
}

export function FolderIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M3 7.5a2 2 0 0 1 2-2h4l2 2H19a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
    </BaseIcon>
  );
}

export function BoardIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <rect x="4" y="5" width="16" height="14" rx="2" />
      <path d="M9 5v14M15 9v10" />
    </BaseIcon>
  );
}

export function ChartIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M5 19V9" />
      <path d="M12 19V5" />
      <path d="M19 19v-7" />
      <path d="M3 19h18" />
    </BaseIcon>
  );
}

export function PulseIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M3 12h4l2-4 4 8 2-4h6" />
    </BaseIcon>
  );
}

export function SprintIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M4 15c1.5-4 4-6 7.5-6H15" />
      <path d="M12 9h4l-1.5-2.5" />
      <path d="M7 16.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0Z" />
      <path d="M20 8a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0Z" />
    </BaseIcon>
  );
}

export function UsersIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M16 19v-1a3 3 0 0 0-3-3H8a3 3 0 0 0-3 3v1" />
      <circle cx="10.5" cy="9" r="3" />
      <path d="M19 19v-1a3 3 0 0 0-2-2.82" />
      <path d="M15.5 6.4a3 3 0 0 1 0 5.2" />
    </BaseIcon>
  );
}

export function StoryIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <rect x="5" y="4" width="14" height="16" rx="2" />
      <path d="M8 8h8M8 12h8M8 16h5" />
    </BaseIcon>
  );
}

export function CrownIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="m4 18 1.5-9 4.5 4 2-5 2 5 4.5-4L20 18Z" />
      <path d="M4 18h16" />
    </BaseIcon>
  );
}

export function SparklesIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="m12 3 1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5Z" />
      <path d="m19 14 .7 2.3L22 17l-2.3.7L19 20l-.7-2.3L16 17l2.3-.7Z" />
      <path d="m5 14 .5 1.5L7 16l-1.5.5L5 18l-.5-1.5L3 16l1.5-.5Z" />
    </BaseIcon>
  );
}

export function InboxIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M4 13V7a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v6" />
      <path d="M4 13h4l2 3h4l2-3h4" />
      <path d="M4 13v4a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-4" />
    </BaseIcon>
  );
}

export function WrenchIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M14 6a4 4 0 0 0 4.7 4.7l-8.4 8.4a2 2 0 1 1-2.8-2.8l8.4-8.4A4 4 0 0 0 14 6Z" />
      <path d="m15 5 4 4" />
    </BaseIcon>
  );
}

export function BugIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M9 7.5V5.8a3 3 0 0 1 6 0v1.7" />
      <path d="M8 10.5h8v4.2a4 4 0 0 1-8 0z" />
      <path d="M4.5 10.5h3M16.5 10.5h3" />
      <path d="M5.5 15h2.8M15.7 15h2.8" />
      <path d="M6.8 6.8 8.6 8M17.2 6.8 15.4 8" />
    </BaseIcon>
  );
}

export function CheckCircleIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <circle cx="12" cy="12" r="9" />
      <path d="m8.5 12 2.5 2.5 4.5-5" />
    </BaseIcon>
  );
}

export function ArchiveIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <rect x="4" y="5" width="16" height="4" rx="1" />
      <path d="M6 9h12v8a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2z" />
      <path d="M10 13h4" />
    </BaseIcon>
  );
}

export function CompassIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <circle cx="12" cy="12" r="8" />
      <path d="m14.8 9.2-1.6 4.6-4.6 1.6 1.6-4.6z" />
    </BaseIcon>
  );
}

export function ClipboardIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <rect x="6" y="5" width="12" height="15" rx="2" />
      <path d="M9 5.5h6" />
      <path d="M9 10h6M9 14h6" />
    </BaseIcon>
  );
}

export function CodeIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="m9 8-4 4 4 4" />
      <path d="m15 8 4 4-4 4" />
      <path d="m13.5 5.5-3 13" />
    </BaseIcon>
  );
}

export function SearchIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <circle cx="11" cy="11" r="5.5" />
      <path d="m16 16 3.5 3.5" />
    </BaseIcon>
  );
}

export function XIcon(props: IconProps) {
  return (
    <BaseIcon {...props}>
      <path d="M6 6l12 12" />
      <path d="M18 6 6 18" />
    </BaseIcon>
  );
}
