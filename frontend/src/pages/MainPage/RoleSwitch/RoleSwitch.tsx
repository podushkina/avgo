import BuyerIllustration from '@/assets/buyer.svg?react';
import SellerIllustration from '@/assets/seller.svg?react';
import { Switch } from '@/components/ui/switch';
import { cn } from '@/lib/utils';
import type { Role } from '@/stores';

type RoleSwitchProps = {
  value: Role;
  onChange: (role: Role) => void;
};

const ROLES = [
  { id: 'buyer', label: 'Я покупатель', Illustration: BuyerIllustration },
  { id: 'seller', label: 'Я продавец', Illustration: SellerIllustration },
] as const;

const RoleSwitch = ({ value, onChange }: RoleSwitchProps) => {
  const labelClassName = (role: Role) =>
    cn(
      'rounded-md text-xl transition-colors outline-none focus-visible:ring-3 focus-visible:ring-ring/50',
      value === role
        ? 'font-extrabold text-foreground'
        : 'font-medium text-muted-foreground hover:text-foreground/70',
    );

  return (
    <div className="flex flex-col items-center gap-8">
      <div className="grid w-full max-w-md grid-cols-[1fr_auto_1fr] items-center gap-4">
        <button
          type="button"
          onClick={() => onChange('buyer')}
          className={cn(labelClassName('buyer'), 'justify-self-end text-right')}
        >
          {ROLES[0].label}
        </button>

        <Switch
          size="lg"
          checked={value === 'seller'}
          onCheckedChange={(checked) => onChange(checked ? 'seller' : 'buyer')}
          aria-label="Переключить роль"
          className="justify-self-center"
        />

        <button
          type="button"
          onClick={() => onChange('seller')}
          className={cn(
            labelClassName('seller'),
            'justify-self-start text-left',
          )}
        >
          {ROLES[1].label}
        </button>
      </div>

      <div className="grid w-full grid-cols-2 gap-4 sm:gap-8">
        {ROLES.map(({ id, label, Illustration }) => (
          <button
            key={id}
            type="button"
            onClick={() => onChange(id)}
            aria-label={label}
            aria-pressed={value === id}
            className={cn(
              'flex aspect-[18/13] items-center justify-center rounded-2xl border p-2 transition-all duration-300 outline-none focus-visible:ring-3 focus-visible:ring-ring/50',
              value === id
                ? 'scale-105 border-primary/40 bg-muted/40 opacity-100 ring-2 ring-primary'
                : 'border-border opacity-40 grayscale hover:opacity-70 hover:grayscale-[0.6]',
            )}
          >
            <Illustration className="size-full" />
          </button>
        ))}
      </div>
    </div>
  );
};

export default RoleSwitch;
