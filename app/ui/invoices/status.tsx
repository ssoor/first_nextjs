import { CheckIcon, ClockIcon } from '@heroicons/react/24/outline';
import clsx from 'clsx';

export default function InvoiceStatus({ status }: { status: string }) {
  return (
    <span
      className={clsx(
        'inline-flex items-center rounded-full px-2 py-1 text-xs',
        {
          'bg-gray-100 text-gray-500': status === '即将开始',
          'bg-green-500 text-white': status === '正在进行',
        },
      )}
    >
      {status === '即将开始' ? (
        <>
          即将开始
          <ClockIcon className="ml-1 w-4 text-gray-500" />
        </>
      ) : null}
      {status === '正在进行' ? (
        <>
          正在进行
          <CheckIcon className="ml-1 w-4 text-white" />
        </>
      ) : null}
    </span>
  );
}
